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
	invalidEventReason       = "invalid_event_err"
	invalidBootTimeReason    = "invalid_boot_time"
	invalidBirthdateReason   = "invalid_birthdate"
	invalidDestinationReason = "invalid_destination"

	nonEventReason      = "non_event"
	eventMismatchReason = "event_type_mismatch"
)

// MetricsLogError is an optional interface for errors to implement if the error should be
// logged by prometheus metrics with a certain label.
type MetricsLogError interface {
	ErrorLabel() string
}

type TaggedError interface {
	Tag() Tag
}

type InvalidEventErr struct {
	OriginalErr error
	ErrorTag    Tag
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

	return invalidEventReason
}

func (e InvalidEventErr) Tag() Tag {
	return e.ErrorTag
}

type InvalidBootTimeErr struct {
	OriginalErr error
	ErrorTag    Tag
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
	return invalidBootTimeReason
}

func (e InvalidBootTimeErr) Tag() Tag {
	if e.ErrorTag == Unknown {
		return InvalidBootTime
	}
	return e.ErrorTag
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
	return invalidBirthdateReason
}

type InconsistentIDErr struct {
	IDs []string
}

func (e InconsistentIDErr) Error() string {
	return "inconsistent device id"
}

func (e InconsistentIDErr) Tag() Tag {
	return InconsistentDeviceID
}

type BootDurationErr struct {
	OriginalErr error
	ErrorTag    Tag
	Destination string
	Timestamps  []int64
}

func (e BootDurationErr) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("boot duration error: %v", e.OriginalErr)
	}

	return "boot duration error"
}

func (e BootDurationErr) Tag() Tag {
	return e.ErrorTag
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

	return invalidDestinationReason
}
