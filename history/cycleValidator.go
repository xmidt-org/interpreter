package history

import (
	"errors"
	"fmt"

	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

var (
	ErrInconsistentMetadata = errors.New("inconsistent metadata")
	ErrRepeatID             = errors.New("repeat transaction uuid found")
	ErrMissingOnlineEvent   = errors.New("session does not have online event")
	ErrMissingOfflineEvent  = errors.New("session does not have offline event")
)

// CycleValidatorFunc is a function type that takes in a slice of events
// and returns whether the slice of events is valid or not.
type CycleValidatorFunc func(events []interpreter.Event) (valid bool, err error)

// Valid runs the CycleValidatorFunc.
func (cf CycleValidatorFunc) Valid(events []interpreter.Event) (bool, error) {
	return cf(events)
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
func SessionOnlineValidator(excludeFunc func(id string) bool) CycleValidatorFunc {
	return func(events []interpreter.Event) (bool, error) {
		sessionsWithOnline := parseSessions(events, interpreter.OnlineEventType)
		invalidIds := findSessionsWithoutEvent(sessionsWithOnline, excludeFunc)
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
func SessionOfflineValidator(excludeFunc func(id string) bool) CycleValidatorFunc {
	return func(events []interpreter.Event) (bool, error) {
		if len(events) == 0 {
			return true, nil
		}

		sessionsWithOffline := parseSessions(events, interpreter.OfflineEventType)
		invalidIds := findSessionsWithoutEvent(sessionsWithOffline, excludeFunc)
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

func findSessionsWithoutEvent(eventsMap map[string]bool, exclude func(id string) bool) []string {
	if exclude == nil {
		exclude = func(_ string) bool {
			return false
		}
	}

	var missingEvents []string
	for id, exist := range eventsMap {
		if !exist && !exclude(id) {
			missingEvents = append(missingEvents, id)
		}
	}

	return missingEvents
}

func determineMetadataValues(fields []string, event interpreter.Event) map[string]string {
	values := make(map[string]string)
	for _, field := range fields {
		values[field] = event.Metadata[field]
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
		if event.Metadata[key] != val {
			incorrectMetadata[key] = true
		}
	}

	return incorrectMetadata
}
