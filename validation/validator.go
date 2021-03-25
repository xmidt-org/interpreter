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
	"time"

	"github.com/xmidt-org/interpreter"
)

var (
	ErrInvalidEventType = errors.New("event type doesn't match")
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
func BootTimeValidator(tv TimeValidation) ValidatorFunc {
	return func(e interpreter.Event) (bool, error) {
		bootTime, err := e.BootTime()
		if err != nil || bootTime <= 0 {
			return false, InvalidEventErr{
				OriginalErr: InvalidBootTimeErr{
					OriginalErr: err,
				},
			}
		}

		if valid, err := tv.Valid(time.Unix(bootTime, 0)); !valid {
			return false, InvalidEventErr{
				OriginalErr: InvalidBootTimeErr{
					OriginalErr: err,
				},
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
			return false, InvalidEventErr{
				OriginalErr: InvalidBirthdateErr{},
			}
		}

		if valid, err := tv.Valid(time.Unix(0, e.Birthdate)); !valid {
			return false, InvalidEventErr{
				OriginalErr: InvalidBirthdateErr{
					OriginalErr: err,
				},
			}
		}

		return true, nil
	}
}

// DestinationValidator takes in a regex and returns a ValidatorFunc that checks if an
// Event's destination is valid against the EventRegex and this regex.
func DestinationValidator(regex *regexp.Regexp) ValidatorFunc {
	return func(e interpreter.Event) (bool, error) {
		if !interpreter.EventRegex.MatchString(e.Destination) || !regex.MatchString(e.Destination) {
			return false, InvalidEventErr{OriginalErr: ErrInvalidEventType}
		}
		return true, nil
	}
}
