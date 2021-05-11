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
	"strings"

	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

var (
	errNewerBootTime  = errors.New("newer boot-time found")
	errDuplicateEvent = errors.New("duplicate event found")
)

// Comparator compares two events and returns true if the condition has been matched.
// A Comparator can also return an error when it deems appropriate.
type Comparator interface {
	Compare(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error)
}

// ComparatorFunc is a function that compares two events.
type ComparatorFunc func(interpreter.Event, interpreter.Event) (bool, error)

// Compare runs the ComparatorFunc, making a ComparatorFunc a Comparator
func (c ComparatorFunc) Compare(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error) {
	return c(baseEvent, newEvent)
}

// Comparators are a list of objects that implement the Comparator interface
type Comparators []Comparator

// Compare runs through a list of Comparators and compares two events using
// each comparator. Returns true on the first comparator that matches.
func (c Comparators) Compare(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error) {
	for _, comparator := range c {
		if match, err := comparator.Compare(baseEvent, newEvent); match {
			return true, err
		}
	}
	return false, nil
}

// OlderBootTimeComparator returns a ComparatorFunc to check and see if newEvent's boot-time is
// less than the baseEvent's boot-time. If it is, it returns true and an error.
// OlderBootTimeComparator assumes that newEvent has a valid boot-time
// and does not do any error-checking of newEvent's boot-time.
func OlderBootTimeComparator() ComparatorFunc {
	return func(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error) {
		// baseEvent is newEvent, no need to compare boot-times
		if baseEvent.TransactionUUID == newEvent.TransactionUUID {
			return false, nil
		}

		latestBootTime, _ := newEvent.BootTime()
		bootTime, err := baseEvent.BootTime()
		if err != nil || bootTime <= 0 {
			return false, nil
		}

		// if this event has a boot-time more recent than the latest one, return an error
		if bootTime > latestBootTime {
			return true, ComparatorErr{OriginalErr: errNewerBootTime, ErrorTag: validation.OutdatedBootTime, ComparisonEvent: baseEvent}
		}

		return false, nil

	}
}

// DuplicateEventComparator returns a ComparatorFunc to check and see if newEvent is a duplicate. A duplicate event
// in this case is defined as sharing the same event type and boot-time as the base event while having a birthdate
// that is equal to or newer than baseEvent's birthdate. If newEvent is found to be a duplicate, it returns true and
// an error. It assumes that newEvent has a valid boot-time and event-type and does not do any error-checking
// of newEvent's boot-time or event type.
func DuplicateEventComparator() ComparatorFunc {
	return func(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error) {
		// baseEvent is newEvent, no need to compare boot-times
		if baseEvent.TransactionUUID == newEvent.TransactionUUID {
			return false, nil
		}

		baseEventType, err := baseEvent.EventType()
		if err != nil {
			return false, nil
		}

		newEventType, _ := newEvent.EventType()

		// see if event types match
		if strings.ToLower(strings.TrimSpace(baseEventType)) == strings.ToLower(strings.TrimSpace(newEventType)) {
			latestBootTime, _ := newEvent.BootTime()
			bootTime, err := baseEvent.BootTime()
			if err != nil || bootTime <= 0 {
				return false, nil
			}

			// If the boot-time is the same as the latestBootTime, and the birthdate is older or equal,
			// this means that newEvent is a duplicate.
			if bootTime == latestBootTime && baseEvent.Birthdate <= newEvent.Birthdate {
				return true, ComparatorErr{OriginalErr: errDuplicateEvent, ErrorTag: validation.DuplicateEvent, ComparisonEvent: baseEvent}
			}
		}

		return false, nil
	}
}
