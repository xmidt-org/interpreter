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
	"sort"

	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

var (
	ErrInconsistentMetadata = errors.New("inconsistent metadata")
	ErrRepeatID             = errors.New("repeat transaction uuid found")
	ErrMissingOnlineEvent   = errors.New("session does not have online event")
	ErrMissingOfflineEvent  = errors.New("session does not have offline event")
	ErrInvalidEventOrder    = errors.New("invalid event order")
	ErrFalseReboot          = errors.New("not a true reboot")
	ErrNoReboot             = errors.New("no reboot found")
)

// CycleValidator validates a list of events.
type CycleValidator interface {
	Valid(events []interpreter.Event) (bool, error)
}

// CycleValidatorFunc is a function type that takes in a slice of events
// and returns whether the slice of events is valid or not.
type CycleValidatorFunc func(events []interpreter.Event) (valid bool, err error)

// Valid runs the CycleValidatorFunc.
func (cf CycleValidatorFunc) Valid(events []interpreter.Event) (bool, error) {
	return cf(events)
}

// DefaultCycleValidator is a CycleValidator that always returns true and nil.
func DefaultCycleValidator() CycleValidatorFunc {
	return func(_ []interpreter.Event) (bool, error) {
		return true, nil
	}
}

// CycleValidators are a list of objects that implement the CycleValidator interface
type CycleValidators []CycleValidator

// Valid runs through a list of CycleValidators and checks that the list of events
// is valid against each validator. It runs through all of the validators
// and returns the errors collected from each one. If at least one validator returns
// false, then false is returned.
func (c CycleValidators) Valid(events []interpreter.Event) (bool, error) {
	var allErrors validation.Errors
	for _, validator := range c {
		if valid, err := validator.Valid(events); !valid {
			allErrors = append(allErrors, err)
		}
	}

	if len(allErrors) == 0 {
		return true, nil
	}

	return false, allErrors
}

// MetadataValidator takes in a slice of metadata keys and returns a CycleValidatorFunc that
// validates that events in the slice have the same values for the keys passed in. If
// checkWithinCycle is true, it will only check that events with the same boot-time have the same
// values.
func MetadataValidator(fields []string, checkWithinCycle bool) CycleValidatorFunc {
	return func(events []interpreter.Event) (bool, error) {
		var incorrectFields []string
		if checkWithinCycle {
			incorrectFields = validateMetadataWithinCycle(fields, events)
		} else {
			incorrectFields = validateMetadata(fields, events)
		}

		if len(incorrectFields) == 0 {
			return true, nil
		}

		var err error
		if checkWithinCycle {
			err = fmt.Errorf("%w among same boot-time events", ErrInconsistentMetadata)
		} else {
			err = ErrInconsistentMetadata
		}

		return false, CycleValidationErr{
			OriginalErr:       err,
			ErrorDetailKey:    "inconsistent metadata keys",
			ErrorDetailValues: incorrectFields,
			ErrorTag:          validation.InconsistentMetadata,
		}
	}
}

// TransactionUUIDValidator returns a CycleValidatorFunc that validates that all events in the slice
// have different TransactionUUIDs.
func TransactionUUIDValidator() CycleValidatorFunc {
	return func(events []interpreter.Event) (bool, error) {
		ids := make(map[string]bool)
		for _, event := range events {
			if _, found := ids[event.TransactionUUID]; !found {
				ids[event.TransactionUUID] = false
			} else {
				ids[event.TransactionUUID] = true
			}
		}

		var repeatIDSlice []string
		for id, repeated := range ids {
			if repeated {
				repeatIDSlice = append(repeatIDSlice, id)
			}
		}

		if len(repeatIDSlice) == 0 {
			return true, nil
		}

		return false, CycleValidationErr{
			OriginalErr:       ErrRepeatID,
			ErrorDetailKey:    "repeated uuids",
			ErrorDetailValues: repeatIDSlice,
			ErrorTag:          validation.RepeatedTransactionUUID,
		}
	}
}

// SessionOnlineValidator returns a CycleValidatorFunc that validates that all sessions in the slice
// (determined by sessionIDs) have an online event. It takes in excludeFunc, which is a function that
// takes in a session ID and returns true if that session is still valid even if it does not have an online event.
func SessionOnlineValidator(excludeFunc func(events []interpreter.Event, id string) bool) CycleValidatorFunc {
	return func(events []interpreter.Event) (bool, error) {
		sessionsWithOnline := parseSessions(events, interpreter.OnlineEventType)
		invalidIds := findSessionsWithoutEvent(sessionsWithOnline, events, excludeFunc)
		if len(invalidIds) == 0 {
			return true, nil
		}

		return false, CycleValidationErr{
			OriginalErr:       ErrMissingOnlineEvent,
			ErrorDetailKey:    "session ids",
			ErrorDetailValues: invalidIds,
			ErrorTag:          validation.MissingOnlineEvent,
		}

	}
}

// SessionOfflineValidator returns a CycleValidatorFunc that validates that all sessions in the slice
// (except for the most recent session) have an offline event. It takes in excludeFunc, which is a function that
// takes in a session ID and returns true if that session is still valid even if it does not have an offline event.
func SessionOfflineValidator(excludeFunc func(events []interpreter.Event, id string) bool) CycleValidatorFunc {
	return func(events []interpreter.Event) (bool, error) {
		if len(events) == 0 {
			return true, nil
		}

		sessionsWithOffline := parseSessions(events, interpreter.OfflineEventType)
		invalidIds := findSessionsWithoutEvent(sessionsWithOffline, events, excludeFunc)
		if len(invalidIds) == 0 {
			return true, nil
		}

		return false, CycleValidationErr{
			OriginalErr:       ErrMissingOfflineEvent,
			ErrorDetailKey:    "session ids",
			ErrorDetailValues: invalidIds,
			ErrorTag:          validation.MissingOfflineEvent,
		}

	}
}

// EventOrderValidator returns a CycleValidatorFunc that validates that there exists, within the history of events,
// particular events in the proper order.
func EventOrderValidator(order []string) CycleValidatorFunc {
	return func(events []interpreter.Event) (bool, error) {
		if len(order) == 0 {
			return true, nil
		}

		currentIndex := 0
		validOrder := true
		var actualOrder []string
		for _, event := range events {
			if currentIndex >= len(order) {
				break
			}

			eventType, _ := event.EventType()
			if currentIndex > 0 {
				actualOrder = append(actualOrder, eventType)
				if eventType != order[currentIndex] {
					validOrder = false
				} else {
					currentIndex++
				}
			} else if currentIndex == 0 {
				if eventType == order[currentIndex] {
					actualOrder = append(actualOrder, eventType)
					currentIndex++
				}
			}
		}

		if !validOrder || currentIndex != len(order) {
			return false, CycleValidationErr{
				OriginalErr:       ErrInvalidEventOrder,
				ErrorDetailKey:    "event_order",
				ErrorDetailValues: actualOrder,
				ErrorTag:          validation.InvalidEventOrder,
			}
		}

		return true, nil
	}
}

// TrueRebootValidator returns a CycleValidatorFunc that validates that the latest online event is the result of a
// true reboot, meaning that it has a boot-time that is different from the event that precedes it.
// If an online event is not found, false and an error is returned.
func TrueRebootValidator() CycleValidatorFunc {
	return func(events []interpreter.Event) (bool, error) {
		eventsCopy := make([]interpreter.Event, len(events))
		copy(eventsCopy, events)
		sort.Slice(eventsCopy, func(a, b int) bool {
			boottimeA, _ := eventsCopy[a].BootTime()
			boottimeB, _ := eventsCopy[b].BootTime()
			if boottimeA != boottimeB {
				return boottimeA > boottimeB
			}

			return eventsCopy[a].Birthdate > eventsCopy[b].Birthdate
		})

		for i, event := range eventsCopy {
			eventType, _ := event.EventType()
			if eventType == interpreter.OnlineEventType {
				if i < len(eventsCopy)-1 {
					nextEvent := eventsCopy[i+1]
					currentBootTime, err := event.BootTime()
					nextEventBootTime, e := nextEvent.BootTime()
					if err != nil || e != nil || currentBootTime == nextEventBootTime {
						return false, CycleValidationErr{
							OriginalErr: ErrFalseReboot,
							ErrorTag:    validation.FalseReboot,
						}
					}
				}

				return true, nil
			}
		}

		return false, CycleValidationErr{
			OriginalErr: ErrNoReboot,
			ErrorTag:    validation.NoReboot,
		}
	}
}

// go through list of events and save all session ids seen in the list as well as whether that session
// has the event being looked for.
func parseSessions(events []interpreter.Event, searchedEventType string) map[string]bool {
	eventsMap := make(map[string]bool)
	for _, event := range events {
		sessionID := event.SessionID
		eventType, err := event.EventType()
		if len(sessionID) == 0 || err != nil {
			continue
		}

		if _, found := eventsMap[sessionID]; !found {
			eventsMap[sessionID] = false
		}

		if eventType == searchedEventType {
			eventsMap[sessionID] = true
		}

	}
	return eventsMap
}

func findSessionsWithoutEvent(eventsMap map[string]bool, eventsList []interpreter.Event, exclude func(events []interpreter.Event, id string) bool) []string {
	if exclude == nil {
		exclude = func(_ []interpreter.Event, _ string) bool {
			return false
		}
	}

	var missingEvents []string
	for id, exist := range eventsMap {
		if !exist && !exclude(eventsList, id) {
			missingEvents = append(missingEvents, id)
		}
	}

	return missingEvents
}

func determineMetadataValues(fields []string, event interpreter.Event) map[string]string {
	values := make(map[string]string)
	for _, field := range fields {
		values[field], _ = event.GetMetadataValue(field)
	}

	return values
}

func validateMetadata(keys []string, events []interpreter.Event) []string {
	if len(events) == 0 {
		return nil
	}

	// save what the metadata values are supposed to be for all following events
	metadataVals := determineMetadataValues(keys, events[0])
	incorrectFieldsMap := make(map[string]bool)
	for _, event := range events {
		// check that each event's metadata values are what they are supposed to be
		incorrectFieldsMap = checkMetadataValues(metadataVals, incorrectFieldsMap, event)
	}

	if len(incorrectFieldsMap) == 0 {
		return nil
	}

	fields := make([]string, 0, len(incorrectFieldsMap))
	for key := range incorrectFieldsMap {
		fields = append(fields, key)
	}

	return fields

}

// validate that metdata is the same within events with the same boot-time
func validateMetadataWithinCycle(keys []string, events []interpreter.Event) []string {
	if len(events) == 0 {
		return nil
	}

	// map saving the metadata values that all events with a certain boot-time must have
	metadataVals := make(map[int64]map[string]string)
	incorrectFieldsMap := make(map[string]bool)
	for _, event := range events {
		boottime, err := event.BootTime()
		if err != nil || boottime <= 0 {
			continue
		}

		expectedVals, found := metadataVals[boottime]
		// if metadata values for that boot-time does not exist, this is the first time we've encountered
		// an event with this boot-time, so find the values of the metadata keys and save them in the map
		// to reference later.
		if !found {
			metadataVals[boottime] = determineMetadataValues(keys, event)
			continue
		}

		// compare the event's metadata values to the correct metadata values.
		incorrectFieldsMap = checkMetadataValues(expectedVals, incorrectFieldsMap, event)
	}

	if len(incorrectFieldsMap) == 0 {
		return nil
	}

	fields := make([]string, 0, len(incorrectFieldsMap))
	for key := range incorrectFieldsMap {
		fields = append(fields, key)
	}

	return fields

}

// compare an event's metadata values with the values it is supposed to have
func checkMetadataValues(expectedMetadataVals map[string]string, incorrectMetadata map[string]bool, event interpreter.Event) map[string]bool {
	for key, val := range expectedMetadataVals {
		if eventVal, _ := event.GetMetadataValue(key); eventVal != val {
			incorrectMetadata[key] = true
		}
	}

	return incorrectMetadata
}
