package history

import (
	"github.com/xmidt-org/interpreter"
)

type CycleValidatorFunc func(events []interpreter.Event) (valid bool, err error)

// TODO: implement checkWithinCycle version
func MetadataValidator(fields []string, checkWithinCycle bool) CycleValidatorFunc {
	metadataValues := make(map[string]string)
	incorrectMetadata := make(map[string]bool)
	return func(events []interpreter.Event) (bool, error) {
		for i, event := range events {
			if i == 0 {
				metadataValues = determineMetadataValues(fields, event)
			} else {
				incorrectMetadata = checkMetadataValues(metadataValues, incorrectMetadata, event)
			}
		}

		if len(incorrectMetadata) == 0 {
			return true, nil
		}

		incorrectFields := make([]string, 0, len(incorrectMetadata))
		for field := range incorrectMetadata {
			incorrectFields = append(incorrectFields, field)
		}

		return false, InconsistentMetadataErr{
			InconsistentFields: incorrectFields,
		}
	}
}

func TransactionUUIDValidator() CycleValidatorFunc {
	ids := make(map[string]bool)
	repeatedIds := make(map[string]bool)
	var repeatIDSlice []string
	return func(events []interpreter.Event) (bool, error) {
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

		return false, RepeatIDErr{
			RepeatedIDs: repeatIDSlice,
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

func checkMetadataValues(expectedMetadataVals map[string]string, incorrectMetadata map[string]bool, event interpreter.Event) map[string]bool {
	for key, val := range expectedMetadataVals {
		if v, found := event.Metadata[key]; !found || v != val {
			incorrectMetadata[key] = true
		}
	}

	return incorrectMetadata
}
