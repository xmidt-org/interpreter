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

	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

var (
	EventNotFoundErr = errors.New("event not found")
)

// FinderFunc is a function type that takes in a slice of events
// and the current event and returns an Event from the slice.
type FinderFunc func([]interpreter.Event, interpreter.Event) (interpreter.Event, error)

func (f FinderFunc) Find(events []interpreter.Event, currentEvent interpreter.Event) (interpreter.Event, error) {
	return f(events, currentEvent)
}

// LastSessionFinder returns a function to find an event that is deemed valid by the Validator passed in
// with the boot-time of the previous session.
func LastSessionFinder(validator validation.Validator) FinderFunc {
	return func(events []interpreter.Event, currentEvent interpreter.Event) (interpreter.Event, error) {
		// verify that the current event has a boot-time
		currentBootTime, err := currentEvent.BootTime()
		if currentBootTime <= 0 {
			return interpreter.Event{}, validation.InvalidBootTimeErr{OriginalErr: err}
		}

		event, found := lastSessionFinder(events, currentEvent, validator)
		// final check to make sure that we actually found an event
		if !found {
			return interpreter.Event{}, EventFinderErr{OriginalErr: EventNotFoundErr}
		}
		return event, nil
	}
}

func lastSessionFinder(events []interpreter.Event, currentEvent interpreter.Event, validator validation.Validator) (interpreter.Event, bool) {
	currentBootTime, _ := currentEvent.BootTime()

	var latestEvent interpreter.Event
	var found bool
	var prevBootTime int64

	for _, event := range events {

		// if transaction UUIDs are the same, continue onto next event
		if event.TransactionUUID == currentEvent.TransactionUUID {
			continue
		}

		// figure out the latest previous boot-time
		if eBoot, newTime := getPreviousBootTime(event, prevBootTime, currentBootTime); newTime {
			prevBootTime = eBoot
			found = false
		}

		// if event does not match validators, continue onto next event.
		if eventValid := newEventValid(event, latestEvent, validator, prevBootTime); eventValid {
			latestEvent = event
			found = true
		}
	}

	return latestEvent, found
}

// CurrentSessionFinder returns a function to find an event that is deemed valid by the Validator passed in
// with the boot-time of the current event.
func CurrentSessionFinder(validator validation.Validator) FinderFunc {
	return func(events []interpreter.Event, currentEvent interpreter.Event) (interpreter.Event, error) {
		// verify that the current event has a boot-time
		currentBootTime, err := currentEvent.BootTime()
		if currentBootTime <= 0 {
			return interpreter.Event{}, validation.InvalidBootTimeErr{OriginalErr: err}
		}

		event, found := currentSessionFinder(events, currentEvent, validator)
		// final check to make sure that we actually found an event
		if !found {
			return interpreter.Event{}, EventFinderErr{OriginalErr: EventNotFoundErr}
		}
		return event, nil
	}
}

func currentSessionFinder(events []interpreter.Event, currentEvent interpreter.Event, validator validation.Validator) (interpreter.Event, bool) {
	currentBootTime, _ := currentEvent.BootTime()

	var latestEvent interpreter.Event
	var found bool
	for _, event := range events {
		// if transaction UUIDs are the same, continue onto next event
		if event.TransactionUUID == currentEvent.TransactionUUID {
			continue
		}

		// Get the bootTime from the event we are checking. If boot-time
		// doesn't exist, move on to the next event.
		bootTime, _ := event.BootTime()
		if bootTime <= 0 {
			continue
		}

		// if event does not match validators, continue onto next event.
		if eventValid := newEventValid(event, latestEvent, validator, currentBootTime); eventValid {
			latestEvent = event
			found = true
		}
	}

	return latestEvent, found
}

// See if event has a boot-time that has greater than the one we are currently tracking but less than
// the latestBootTime.
func getPreviousBootTime(event interpreter.Event, currentPrevTime int64, latestBootTime int64) (int64, bool) {
	// Get the bootTime from the event we are checking. If boot-time
	// doesn't exist, return currentPrevTime, which is the latest previous time currently found.
	bootTime, _ := event.BootTime()
	if bootTime <= 0 {
		return currentPrevTime, false
	}

	// if boot-time is greater than any we've found so far but less than the current boot-time,
	// return bootTime
	if bootTime > currentPrevTime && bootTime < latestBootTime {
		return bootTime, true
	}
	return currentPrevTime, false
}

// Sees if an event is valid based on the validators passed in and whether it has the targetBootTime.
func newEventValid(newEvent interpreter.Event, defaultEvent interpreter.Event, validators validation.Validator, targetBootTime int64) bool {
	bootTime, _ := newEvent.BootTime()
	currentPrevBootTime, _ := defaultEvent.BootTime()

	// if boot-time doesn't match target boot-time, return previous event
	if bootTime != targetBootTime {
		return false
	}

	// if event does not match validators, return previous event
	if valid, _ := validators.Valid(newEvent); !valid {
		return false
	}

	if currentPrevBootTime != targetBootTime || newEvent.Birthdate < defaultEvent.Birthdate {
		return true
	}

	return false
}
