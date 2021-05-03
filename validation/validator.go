/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package validation

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/xmidt-org/interpreter"
)

var (
	ErrInvalidEventType     = errors.New("event type doesn't match")
	ErrNonEvent             = errors.New("not an event")
	ErrInconsistentDeviceID = errors.New("inconsistent device id")
	ErrInvalidBootTime      = errors.New("boot-time is past the cut-off year")
	ErrFastBoot             = errors.New("fast booting")
)

// Validator validates an event, returning false and an error if the event is not valid
// and true if the event is valid
type Validator interface {
	Valid(interpreter.Event) (bool, error)
}

// ValidatorFunc is a function that checks if an Event is valid
type ValidatorFunc func(interpreter.Event) (bool, error)

// Valid runs the ValidatorFunc, making a ValidatorFunc a Validator
func (vf ValidatorFunc) Valid(e interpreter.Event) (bool, error) {
	return vf(e)
}

// Validators are a list of objects that implement the Validator interface
type Validators []Validator

// Valid runs through a list of Validators and checks that the Event
// is valid against each validator. Returns false and an error at the first
// validator that deems the Event invalid
func (v Validators) Valid(e interpreter.Event) (bool, error) {
	for _, r := range v {
		if valid, err := r.Valid(e); !valid {
			return false, err
		}
	}
	return true, nil
}

// BootTimeValidator returns a ValidatorFunc that checks if an
// Event's boot-time is valid, meaning parsable, greater than 0, and within the
// bounds deemed valid by the TimeValidation parameter.
func BootTimeValidator(tv TimeValidation, lastValidYear int) ValidatorFunc {
	return func(e interpreter.Event) (bool, error) {
		bootTime, err := e.BootTime()
		if err != nil || bootTime <= 0 {
			var tag Tag
			if errors.Is(err, interpreter.ErrBootTimeNotFound) {
				tag = MissingBootTime
			} else {
				tag = InvalidBootTime
			}

			return false, InvalidBootTimeErr{
				OriginalErr: err,
				ErrorTag:    tag,
			}
		}

		// check that boot-time is after the same day in the desired year
		now := time.Now()
		eventTime := time.Unix(bootTime, 0)
		compareDate := time.Date(lastValidYear, now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

		if eventTime.Before(compareDate) {
			return false, InvalidBootTimeErr{
				OriginalErr: fmt.Errorf("%w: %d", ErrInvalidBootTime, lastValidYear),
				ErrorTag:    InvalidBootTime,
			}
		}

		if valid, err := tv.Valid(time.Unix(bootTime, 0)); !valid {
			var tag Tag
			if errors.Is(err, ErrPastDate) {
				tag = OldBootTime
			} else {
				tag = InvalidBootTime
			}

			return false, InvalidBootTimeErr{
				OriginalErr: err,
				ErrorTag:    tag,
			}
		}

		return true, nil
	}
}

// BirthdateValidator returns a ValidatorFunc that checks if an
// Event's birthdate is valid, meaning greater than 0 and within the
// bounds deemed valid by the TimeValidation parameter.
func BirthdateValidator(tv TimeValidation) ValidatorFunc {
	return func(e interpreter.Event) (bool, error) {
		birthdate := e.Birthdate
		if birthdate <= 0 {
			return false, InvalidBirthdateErr{}
		}

		if valid, err := tv.Valid(time.Unix(0, e.Birthdate)); !valid {
			return false, InvalidBirthdateErr{
				OriginalErr: err,
			}
		}

		return true, nil
	}
}

// DestinationValidator takes in a regex and returns a ValidatorFunc that checks if an
// Event's destination is valid against the EventRegex and this regex.
func DestinationValidator(regex *regexp.Regexp) ValidatorFunc {
	return func(e interpreter.Event) (bool, error) {
		if !interpreter.EventRegex.MatchString(e.Destination) {
			return false, InvalidDestinationErr{
				OriginalErr: ErrNonEvent,
				ErrLabel:    nonEventReason,
			}

		}

		if !regex.MatchString(e.Destination) {
			return false, InvalidDestinationErr{
				OriginalErr: ErrInvalidEventType,
				ErrLabel:    eventMismatchReason,
			}
		}

		return true, nil
	}
}

// ConsistentDeviceIDValidator returns a ValidatorFunc that validates that the all occurrences
// of the device id in an event's source, destination, or metadata are consistent.
func ConsistentDeviceIDValidator() ValidatorFunc {
	return func(e interpreter.Event) (bool, error) {
		var id string
		var consistent bool

		if consistent, id = deviceIDComparison(e.Source, id); !consistent {
			return false, InvalidEventErr{OriginalErr: ErrInconsistentDeviceID, ErrorTag: InconsistentDeviceID}
		}

		if consistent, id = deviceIDComparison(e.Destination, id); !consistent {
			return false, InvalidEventErr{OriginalErr: ErrInconsistentDeviceID, ErrorTag: InconsistentDeviceID}
		}

		for _, val := range e.Metadata {
			if consistent, id = deviceIDComparison(val, id); !consistent {
				return false, InvalidEventErr{OriginalErr: ErrInconsistentDeviceID, ErrorTag: InconsistentDeviceID}
			}
		}

		return true, nil
	}
}

func deviceIDComparison(checkID string, foundID string) (bool, string) {
	if matches := interpreter.DeviceIDRegex.FindAllStringSubmatch(checkID, -1); len(matches) > 0 {
		if len(foundID) == 0 {
			foundID = matches[0][0]
		}

		for _, m := range matches {
			if foundID != m[0] {
				return false, foundID
			}
		}
	}

	return true, foundID
}

// DestinationTimestampValidator returns a ValidatorFunc that validates that the all unix timestamps
// in the destination of an event are at least a certain time duration from the boot-time of the event.
func DestinationTimestampValidator(minDuration time.Duration) ValidatorFunc {
	timestampRegex := regexp.MustCompile(`/(?P<content>[^/]+)`)
	index := timestampRegex.SubexpIndex("content")
	return func(e interpreter.Event) (bool, error) {
		bootTimeInt, err := e.BootTime()
		if err != nil || bootTimeInt <= 0 {
			return true, nil
		}

		bootTime := time.Unix(bootTimeInt, 0)

		matches := timestampRegex.FindAllStringSubmatch(e.Destination, -1)
		for _, match := range matches {
			if val, err := strconv.ParseInt(match[index], 10, 64); err == nil {
				timeStamp := time.Unix(val, 0)
				if bootTime.Before(timeStamp) && timeStamp.Sub(bootTime) < minDuration {
					return false, InvalidEventErr{OriginalErr: ErrFastBoot, ErrorTag: FastBoot}
				}
			}
		}

		return true, nil
	}
}
