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
	"regexp"
	"strconv"
	"time"

	"github.com/xmidt-org/interpreter"
)

var (
	ErrInvalidEventType = errors.New("event type doesn't match")
	ErrNonEvent         = errors.New("not an event")
	ErrFastBoot         = errors.New("fast booting")
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
// Event's boot-time is valid (meaning parsable), greater than 0, and within the
// bounds deemed valid by the TimeValidation parameters.
func BootTimeValidator(tv TimeValidation, yearValidator TimeValidation) ValidatorFunc {
	return func(e interpreter.Event) (bool, error) {
		bootTime, err := getBootTime(e)
		if err != nil {
			return false, err
		}

		if valid, err := yearValidator.Valid(bootTime); !valid {
			return false, InvalidBootTimeErr{
				OriginalErr: err,
				ErrorTag:    InvalidBootTime,
			}
		}

		if valid, err := tv.Valid(bootTime); !valid {
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
		consistent := true
		var firstID string
		var ids []string

		consistent, firstID, ids = consistentIDHelper(e.Source, firstID, consistent, ids)
		consistent, firstID, ids = consistentIDHelper(e.Destination, firstID, consistent, ids)

		for _, val := range e.Metadata {
			consistent, firstID, ids = consistentIDHelper(val, firstID, consistent, ids)
		}

		if !consistent {
			return false, InconsistentIDErr{IDs: ids}
		}

		return true, nil
	}
}

// BootDurationValidator returns a ValidatorFunc that validates that all unix timestamps
// in the destination of an event are at least a certain time duration from the boot-time of the event,
// ensuring that the boot cycle is not suspiciously fast. Note: this validator depends on the boot-time
// being present in an event's metadata. If it isn't, the validator will return true and an error, which
// deems the timestamps as valid, even if they may not be, because it is impossible to determine validity without a boot-time.
func BootDurationValidator(minDuration time.Duration) ValidatorFunc {
	timestampRegex := regexp.MustCompile(`/(?P<content>[^/]+)`)
	index := timestampRegex.SubexpIndex("content")
	return func(e interpreter.Event) (bool, error) {
		bootTime, err := getBootTime(e)
		if err != nil {
			return true, err
		}

		matches := timestampRegex.FindAllStringSubmatch(e.Destination, -1)
		var invalidTimestamps []int64
		valid := true
		for _, match := range matches {
			if val, err := strconv.ParseInt(match[index], 10, 64); err == nil {
				timeStamp := time.Unix(val, 0)
				if bootTime.Before(timeStamp) && timeStamp.Sub(bootTime) < minDuration {
					valid = false
					invalidTimestamps = append(invalidTimestamps, val)
				}
			}
		}

		if !valid {
			return false, BootDurationErr{OriginalErr: ErrFastBoot, Destination: e.Destination, Timestamps: invalidTimestamps, ErrorTag: FastBoot}
		}
		return true, nil
	}
}

func consistentIDHelper(strToCheck string, compareID string, overallConsistent bool, allIDs []string) (bool, string, []string) {
	consistent, foundID, ids := deviceIDComparison(strToCheck, compareID, allIDs)

	allConsistent := consistent && overallConsistent
	return allConsistent, foundID, ids
}

func deviceIDComparison(strToCheck string, compareID string, ids []string) (bool, string, []string) {
	consistent := true
	if matches := interpreter.DeviceIDRegex.FindAllStringSubmatch(strToCheck, -1); len(matches) > 0 {
		if len(compareID) == 0 {
			compareID = matches[0][0]
		}

		for _, m := range matches {
			ids = append(ids, m[0])
			if compareID != m[0] {
				consistent = false
			}
		}
	}

	return consistent, compareID, ids
}

// helper function for getting boot-time and returning the proper error if there is trouble
// getting it.
func getBootTime(e interpreter.Event) (time.Time, error) {
	bootTimeInt, err := e.BootTime()
	if err != nil || bootTimeInt <= 0 {
		var tag Tag
		if errors.Is(err, interpreter.ErrBootTimeNotFound) {
			tag = MissingBootTime
		} else {
			tag = InvalidBootTime
		}

		return time.Time{}, InvalidBootTimeErr{
			OriginalErr: err,
			ErrorTag:    tag,
		}
	}

	return time.Unix(bootTimeInt, 0), nil
}
