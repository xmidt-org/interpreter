package history

import (
	"errors"

	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

var (
	ErrInconsistentMetadata = errors.New("inconsistent metadata")
	ErrRepeatID             = errors.New("repeat transaction uuid found")
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

		return false, CycleValidationErr{
			OriginalErr:   ErrInconsistentMetadata,
			InvalidFields: incorrectFields,
			ErrorTag:      validation.InconsistentMetadata,
		}
	}
}

// TransactionUUIDValidator returns a CycleValidatorFunc that validates that all events in the slice
// have different TransactionUUIDs.
func TransactionUUIDValidator() CycleValidatorFunc {
	return func(events []interpreter.Event) (bool, error) {
		ids := make(map[string]bool)
		repeatedIds := make(map[string]bool)
		var repeatIDSlice []string
		for _, event := range events {
			if ids[event.TransactionUUID] {
				if !repeatedIds[event.TransactionUUID] {
					repeatIDSlice = append(repeatIDSlice, event.TransactionUUID)
					repeatedIds[event.TransactionUUID] = true
				}
			} else {
				ids[event.TransactionUUID] = true
			}
		}

		if len(repeatedIds) == 0 {
			return true, nil
		}

		return false, CycleValidationErr{
			OriginalErr:   ErrRepeatID,
			InvalidFields: repeatIDSlice,
			ErrorTag:      validation.RepeatedTransactionUUID,
		}
	}

}

func determineMetadataValues(fields []string, event interpreter.Event) map[string]string {
	values := make(map[string]string)
	for _, field := range fields {
		if val, ok := event.Metadata[field]; ok {
			values[field] = val
		} else {
			values[field] = ""
		}
	}

	return values
}

func validateMetadata(keys []string, events []interpreter.Event) []string {
	if len(events) == 0 {
		return nil
	}

	metadataVals := determineMetadataValues(keys, events[0])
	incorrectFieldsMap := make(map[string]bool)
	for _, event := range events {
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

func validateMetadataWithinCycle(keys []string, events []interpreter.Event) []string {
	if len(events) == 0 {
		return nil
	}

	metadataVals := make(map[int64]map[string]string)
	incorrectFieldsMap := make(map[string]bool)
	for _, event := range events {
		boottime, err := event.BootTime()
		if err != nil || boottime <= 0 {
			continue
		}

		expectedVals, found := metadataVals[boottime]
		if !found {
			expectedVals = determineMetadataValues(keys, event)
			metadataVals[boottime] = expectedVals
			continue
		}

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

func checkMetadataValues(expectedMetadataVals map[string]string, incorrectMetadata map[string]bool, event interpreter.Event) map[string]bool {
	for key, val := range expectedMetadataVals {
		v, found := event.Metadata[key]
		if found && v != val {
			incorrectMetadata[key] = true
		} else if !found && len(val) > 0 {
			incorrectMetadata[key] = true
		}
	}

	return incorrectMetadata
}
