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
	ErrInvalidEventType     = errors.New("event type is not valid")
	ErrEventTypeMismatch    = errors.New("event type doesn't match")
	ErrNonEvent             = errors.New("not an event")
	ErrFastBoot             = errors.New("fast booting")
	ErrEventRegex           = errors.New("event regex does not include type")
	ErrBirthdateDestination = errors.New("birthdate and destination timestamps do not align")
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
// is valid against each validator. It runs through all of the validators
// and returns the errors collected from each one. If at least one validator returns
// false, then false is returned.
func (v Validators) Valid(e interpreter.Event) (bool, error) {
	var allErrors Errors
	for _, r := range v {
		if valid, err := r.Valid(e); !valid {
			allErrors = append(allErrors, err)
		}
	}

	if len(allErrors) == 0 {
		return true, nil
	}

	return false, allErrors
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
			return false, InvalidBirthdateErr{ErrorTag: InvalidBirthdate}
		}

		if valid, err := tv.Valid(time.Unix(0, e.Birthdate)); !valid {
			return false, InvalidBirthdateErr{
				OriginalErr: err,
				ErrorTag:    InvalidBirthdate,
			}
		}

		return true, nil
	}
}

// BirthdateAlignmentValidator returns a ValidatorFunc that validates that the birthdate is within a certain
// bounds of the timestamps in the event destination (if available).
func BirthdateAlignmentValidator(maxDuration time.Duration) ValidatorFunc {
	timestampRegex := regexp.MustCompile(`/(?P<content>[^/]+)`)
	index := timestampRegex.SubexpIndex("content")
	return func(e interpreter.Event) (bool, error) {
		matches := timestampRegex.FindAllStringSubmatch(e.Destination, -1)
		birthdate := time.Unix(0, e.Birthdate)
		var invalidTimestamps []int64
		valid := true
		for _, match := range matches {
			if val, err := strconv.ParseInt(match[index], 10, 64); err == nil {
				timeStamp := time.Unix(val, 0)
				difference := birthdate.Sub(timeStamp)
				if difference < 0 {
					difference = difference * -1
				}

				if difference > maxDuration {
					valid = false
					invalidTimestamps = append(invalidTimestamps, val)
				}
			}
		}

		if !valid {
			return false, InvalidBirthdateErr{
				OriginalErr: ErrBirthdateDestination,
				Destination: e.Destination,
				Timestamps:  invalidTimestamps,
				ErrorTag:    MisalignedBirthdate,
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
				Destination: e.Destination,
				ErrorTag:    NonEvent,
			}
		}

		if !regex.MatchString(e.Destination) {
			return false, InvalidDestinationErr{
				OriginalErr: ErrEventTypeMismatch,
				Destination: e.Destination,
				ErrorTag:    EventTypeMismatch,
			}
		}

		return true, nil
	}
}

// ConsistentDeviceIDValidator returns a ValidatorFunc that validates that all occurrences
// of the device id in an event's source, destination, or metadata are consistent.
func ConsistentDeviceIDValidator() ValidatorFunc {
	return func(e interpreter.Event) (bool, error) {
		consistent := true
		var firstID string
		ids := make(map[string]bool)

		consistent, firstID, ids = consistentIDHelper(e.Source, firstID, consistent, ids)
		consistent, firstID, ids = consistentIDHelper(e.Destination, firstID, consistent, ids)

		for _, val := range e.Metadata {
			consistent, firstID, ids = consistentIDHelper(val, firstID, consistent, ids)
		}

		if !consistent {
			var idArray []string
			for key := range ids {
				idArray = append(idArray, key)
			}
			return false, InconsistentIDErr{IDs: idArray}
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

// EventTypeValidator returns a ValidatorFunc that validates that the event-type provided in the destination
// matches one of the possible outcomes.
func EventTypeValidator() ValidatorFunc {
	return func(e interpreter.Event) (bool, error) {
		eventType, err := e.EventType()
		if err != nil {
			return false, InvalidDestinationErr{
				OriginalErr: err,
				Destination: e.Destination,
				EventType:   eventType,
				ErrorTag:    InvalidEventType,
			}
		}

		if len(eventType) == 0 || !interpreter.EventTypes[eventType] {
			return false, InvalidDestinationErr{
				OriginalErr: ErrInvalidEventType,
				Destination: e.Destination,
				EventType:   eventType,
				ErrorTag:    InvalidEventType,
			}
		}

		return true, nil
	}
}

func consistentIDHelper(strToCheck string, compareID string, overallConsistent bool, allIDs map[string]bool) (bool, string, map[string]bool) {
	consistent, foundID, ids := deviceIDComparison(strToCheck, compareID, allIDs)

	allConsistent := consistent && overallConsistent
	return allConsistent, foundID, ids
}

func deviceIDComparison(strToCheck string, compareID string, ids map[string]bool) (bool, string, map[string]bool) {
	consistent := true
	if matches := interpreter.DeviceIDRegex.FindAllStringSubmatch(strToCheck, -1); len(matches) > 0 {
		if len(compareID) == 0 {
			compareID = matches[0][0]
		}

		for _, m := range matches {
			ids[m[0]] = true
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
