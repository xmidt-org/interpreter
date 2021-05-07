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

// Errors is a Multierror that also acts as an error, so that a log-friendly
// string can be returned but each error in the list can also be accessed.
type Errors []error

// Error concatenates the list of error strings to provide a single string
// that can be used to represent the errors that occurred.
func (e Errors) Error() string {
	var output strings.Builder
	output.Write([]byte("multiple errors: ["))
	for i, msg := range e {
		if i > 0 {
			output.WriteRune(',')
			output.WriteRune(' ')
		}
		output.WriteString(msg.Error())
	}
	output.WriteRune(']')
	return output.String()
}

// Errors returns the list of errors.
func (e Errors) Errors() []error {
	return e
}

// TaggedError is an optional interface for errors to implement if the error should include a Tag.
type TaggedError interface {
	Tag() Tag
}

// InvalidEventErr is a Tag Error that wraps an underlying error.
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

// Tag returns the ErrorTag if it has been set, then checks the underlying error for a tag and returns that if set.
func (e InvalidEventErr) Tag() Tag {
	if e.ErrorTag != Unknown {
		return e.ErrorTag
	}

	var taggedErr TaggedError
	if e.OriginalErr != nil && errors.As(e.OriginalErr, &taggedErr) {
		return taggedErr.Tag()
	}

	return e.ErrorTag
}

// InvalidBootTimeErr is an error returned when the boot-time of an event is invalid.
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

// Tag returns the default tag of InvalidBootTime if the tag is not set.
func (e InvalidBootTimeErr) Tag() Tag {
	if e.ErrorTag == Unknown {
		return InvalidBootTime
	}
	return e.ErrorTag
}

// InvalidBirthdateErr is an error returned when the birthdate of an event is invalid.
type InvalidBirthdateErr struct {
	OriginalErr error
	ErrorTag    Tag
	Destination string
	Timestamps  []int64
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

// Tag returns the InvalidBirthdate tag as default if the tag is not set.
func (e InvalidBirthdateErr) Tag() Tag {
	if e.ErrorTag == Unknown {
		return InvalidBirthdate
	}
	return e.ErrorTag
}

// InconsistentIDErr is an error returned when the ids in an event is inconsistent.
type InconsistentIDErr struct {
	IDs []string
}

func (e InconsistentIDErr) Error() string {
	return "inconsistent device id"
}

// Tag will always return the InconsistentDeviceID tag.
func (e InconsistentIDErr) Tag() Tag {
	return InconsistentDeviceID
}

// BootDurationErr is an error that is returned when the device boot duration is deemed invalid.
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

func (e BootDurationErr) Unwrap() error {
	return e.OriginalErr
}

// Tag returns InvalidBootDuration as the default tag if the tag is not set.
func (e BootDurationErr) Tag() Tag {
	if e.ErrorTag == Unknown {
		return InvalidBootDuration
	}
	return e.ErrorTag
}

// InvalidDestinationErr is an error that is returned whenever there is something wrong with an event's destination.
type InvalidDestinationErr struct {
	OriginalErr error
	ErrorTag    Tag
	Destination string
	EventType   string
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

// Tag returns InvalidDestination as the default tag if the tag is not set.
func (e InvalidDestinationErr) Tag() Tag {
	if e.ErrorTag == Unknown {
		return InvalidDestination
	}
	return e.ErrorTag
}
