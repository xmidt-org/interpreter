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

package history

import (
	"errors"
	"fmt"

	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

// ComparatorErr is used when an error is found with a trigger event
// when comparing it to a another event in the history of events.
type ComparatorErr struct {
	OriginalErr     error
	ComparisonEvent interpreter.Event
	ErrorTag        validation.Tag
}

func (e ComparatorErr) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("comparator error: %v", e.OriginalErr)
	}

	return "comparator error"
}

func (e ComparatorErr) Unwrap() error {
	return e.OriginalErr
}

// Tag implements the TaggedError interface.
func (e ComparatorErr) Tag() validation.Tag {
	if e.ErrorTag != validation.Unknown {
		return e.ErrorTag
	}

	var taggedErr validation.TaggedError
	if e.OriginalErr != nil && errors.As(e.OriginalErr, &taggedErr) {
		return taggedErr.Tag()
	}

	return e.ErrorTag
}

// Event returns the event in history that caused the error to be thrown.
func (e ComparatorErr) Event() interpreter.Event {
	return e.ComparisonEvent
}

// EventFinderErr is an error used by EventFinder.
type EventFinderErr struct {
	OriginalErr error
	ErrorTag    validation.Tag
}

func (e EventFinderErr) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("failed to find event: %v", e.OriginalErr)
	}

	return "failed to find event"
}

func (e EventFinderErr) Unwrap() error {
	return e.OriginalErr
}

// Tag implements the TaggedError interface.
func (e EventFinderErr) Tag() validation.Tag {
	if e.ErrorTag != validation.Unknown {
		return e.ErrorTag
	}

	var taggedErr validation.TaggedError
	if e.OriginalErr != nil && errors.As(e.OriginalErr, &taggedErr) {
		return taggedErr.Tag()
	}

	return e.ErrorTag
}

// CycleValidationErr is an error returned by validators for list of events.
type CycleValidationErr struct {
	OriginalErr   error
	ErrorTag      validation.Tag
	InvalidFields []string
}

func (e CycleValidationErr) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("cycle validation error: %v", e.OriginalErr)
	}

	return "cycle validation error"
}

// Tag implements the TaggedError interface.
func (e CycleValidationErr) Tag() validation.Tag {
	if e.ErrorTag != validation.Unknown {
		return e.ErrorTag
	}

	var taggedErr validation.TaggedError
	if e.OriginalErr != nil && errors.As(e.OriginalErr, &taggedErr) {
		return taggedErr.Tag()
	}

	return e.ErrorTag
}

func (e CycleValidationErr) Unwrap() error {
	return e.OriginalErr
}

// Fields returns the fields that resulted in the error.
func (e CycleValidationErr) Fields() []string {
	return e.InvalidFields
}
