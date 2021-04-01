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
	"strings"
)

const (
	invalidEventLabel       = "invalid_event_err"
	invalidBootTimeLabel    = "invalid_boot_time"
	invalidBirthdateLabel   = "invalid_birthdate"
	invalidDestinationLabel = "invalid_destination"

	nonEventLabel      = "non_event"
	eventMismatchLabel = "event_type_mismatch"
)

// MetricsLogError is an optional interface for errors to implement if the error should be
// logged by prometheus metrics with a certain label.
type MetricsLogError interface {
	ErrorLabel() string
}

type InvalidEventErr struct {
	OriginalErr error
}

func (e InvalidEventErr) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("event invalid: %v", e.OriginalErr)
	}
	return "event invalid"
}

func (e InvalidEventErr) Unwrap() error {
	return e.OriginalErr
}

func (e InvalidEventErr) ErrorLabel() string {
	var err MetricsLogError
	if ok := errors.As(e.OriginalErr, &err); ok {
		return strings.Replace(err.ErrorLabel(), " ", "_", -1)
	}

	return invalidEventLabel
}

type InvalidBootTimeErr struct {
	OriginalErr error
}

func (e InvalidBootTimeErr) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("boot-time invalid: %v", e.OriginalErr)
	}
	return "boot-time invalid"
}

func (e InvalidBootTimeErr) Unwrap() error {
	return e.OriginalErr
}

func (e InvalidBootTimeErr) ErrorLabel() string {
	return invalidBootTimeLabel
}

type InvalidBirthdateErr struct {
	OriginalErr error
}

func (e InvalidBirthdateErr) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("birthdate invalid: %v", e.OriginalErr)
	}
	return "birthdate invalid"
}

func (e InvalidBirthdateErr) Unwrap() error {
	return e.OriginalErr
}

func (e InvalidBirthdateErr) ErrorLabel() string {
	return invalidBirthdateLabel
}

type InvalidDestinationErr struct {
	OriginalErr error
	ErrLabel    string
}

func (e InvalidDestinationErr) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("invalid destination: %v", e.OriginalErr)
	}
	return "invalid destination"
}

func (e InvalidDestinationErr) Unwrap() error {
	return e.OriginalErr
}

func (e InvalidDestinationErr) ErrorLabel() string {
	if len(e.ErrLabel) > 0 {
		return strings.ReplaceAll(e.ErrLabel, " ", "_")
	}

	return invalidDestinationLabel
}
