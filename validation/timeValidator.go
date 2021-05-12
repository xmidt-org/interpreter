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
	"time"
)

var (
	ErrFutureDate  = errors.New("date is too far in the future")
	ErrPastDate    = errors.New("date is too far in the past")
	ErrInvalidYear = errors.New("date is before desired year")
	ErrNilTimeFunc = errors.New("current-time function has not been set")
)

// TimeValidation sees if a given time is within the time frame it is set to validate.
type TimeValidation interface {
	Valid(time.Time) (bool, error)
}

// TimeValidator implements the TimeValidation interface and makes sure that times are in a certain time frame.
type TimeValidator struct {
	Current      func() time.Time
	ValidFrom    time.Duration // should be a negative duration. If not, it will be changed to negative once Valid is called
	ValidTo      time.Duration
	MinValidYear int
	MaxValidYear int
}

// Valid sees if a date is within a time validator's allowed time frame.
func (t TimeValidator) Valid(date time.Time) (bool, error) {
	if t.Current == nil {
		return false, ErrNilTimeFunc
	}

	if date.Before(time.Unix(0, 0)) || date.Equal(time.Unix(0, 0)) {
		return false, ErrPastDate
	}

	if t.ValidFrom.Seconds() > 0 {
		t.ValidFrom = -1 * t.ValidFrom
	}

	now := t.Current()

	// Check if date is before current date in the MinValidYear
	if t.MinValidYear > 0 {
		compareDate := time.Date(t.MinValidYear, now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		if date.Before(compareDate) {
			return false, fmt.Errorf("%w. Year: %d", ErrInvalidYear, t.MinValidYear)
		}
	}

	// Check if date is after current date in the MaxValidYear
	if t.MaxValidYear > 0 {
		compareDate := time.Date(t.MaxValidYear, now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		if date.After(compareDate) {
			return false, fmt.Errorf("%w. Year: %d", ErrInvalidYear, t.MaxValidYear)
		}
	}

	pastTime := now.Add(t.ValidFrom)
	futureTime := now.Add(t.ValidTo)

	if !(pastTime.Before(date) || pastTime.Equal(date)) {
		return false, ErrPastDate
	}

	if !(futureTime.Equal(date) || futureTime.After(date)) {
		return false, ErrFutureDate
	}

	return true, nil
}
