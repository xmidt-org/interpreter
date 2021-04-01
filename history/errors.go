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
	"fmt"
	"strings"

	"github.com/xmidt-org/interpreter"
)

const (
	comparatorErrLabel = "comparator_err"
	finderErrLabel     = "event_finder_err"
)

// ComparatorErr is used when an error is found with a trigger event
// when comparing it to a another event in the history of events.
type ComparatorErr struct {
	OriginalErr     error
	ComparisonEvent interpreter.Event
	ErrLabel        string
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

func (e ComparatorErr) ErrorLabel() string {
	if len(e.ErrLabel) > 0 {
		return strings.ReplaceAll(e.ErrLabel, " ", "_")
	}

	return comparatorErrLabel
}

// Event returns the event in history that caused the error to be thrown.
func (e ComparatorErr) Event() interpreter.Event {
	return e.ComparisonEvent
}

// EventFinderErr is an error used by EventFinder.
type EventFinderErr struct {
	OriginalErr error
	ErrLabel    string
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

func (e EventFinderErr) ErrorLabel() string {
	if len(e.ErrLabel) > 0 {
		return strings.ReplaceAll(e.ErrLabel, " ", "_")
	}

	return finderErrLabel
}
