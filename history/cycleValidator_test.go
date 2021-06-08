package history

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/xmidt-org/interpreter/validation"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/interpreter"
)

func TestMetadataValidator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)

	bootTime1 := fmt.Sprint(now.Unix())
	bootTime2 := fmt.Sprint(now.Add(time.Hour).Unix())

	tests := []struct {
		description           string
		withinCycle           bool
		fields                []string
		events                []interpreter.Event
		expectedValid         bool
		expectedInvalidFields []string
	}{
		{
			description: "valid-whole list",
			withinCycle: false,
			fields:      []string{"test", "test1"},
			events: []interpreter.Event{
				interpreter.Event{
					Metadata: map[string]string{
						"test":  "test",
						"test1": "test1",
						"test2": "test2",
					},
				},
				interpreter.Event{
					Metadata: map[string]string{
						"test":  "test",
						"test1": "test1",
						"test2": "test3",
					},
				},
				interpreter.Event{
					Metadata: map[string]string{
						"test":  "test",
						"test1": "test1",
					},
				},
			},
			expectedValid: true,
		},
		{
			description: "invalid-whole list",
			withinCycle: false,
			fields:      []string{"test", "test1"},
			events: []interpreter.Event{
				interpreter.Event{
					Metadata: map[string]string{
						"test":  "test",
						"test1": "test1",
						"test2": "test2",
					},
				},
				interpreter.Event{
					Metadata: map[string]string{
						"test":  "test",
						"test2": "test3",
					},
				},
				interpreter.Event{
					Metadata: map[string]string{
						"test":  "test1",
						"test1": "test1",
					},
				},
			},
			expectedValid:         false,
			expectedInvalidFields: []string{"test", "test1"},
		},
		{
			description: "valid-within cycle",
			withinCycle: true,
			fields:      []string{"test", "test1"},
			events: []interpreter.Event{
				interpreter.Event{
					Metadata: map[string]string{
						"test":                  "test",
						"test1":                 "test1",
						"test2":                 "test2",
						interpreter.BootTimeKey: bootTime1,
					},
				},
				interpreter.Event{
					Metadata: map[string]string{
						"test":                  "test",
						"test1":                 "test1",
						interpreter.BootTimeKey: bootTime1,
					},
				},
				interpreter.Event{
					Metadata: map[string]string{
						"test":                  "test2",
						"test1":                 "test3",
						"test4":                 "test4",
						interpreter.BootTimeKey: bootTime2,
					},
				},
				interpreter.Event{
					Metadata: map[string]string{
						"test":                  "test2",
						"test1":                 "test3",
						interpreter.BootTimeKey: bootTime2,
					},
				},
			},
			expectedValid: true,
		},
		{
			description: "valid-within cycle",
			withinCycle: true,
			fields:      []string{"test", "test1"},
			events: []interpreter.Event{
				interpreter.Event{
					Metadata: map[string]string{
						"test":                  "test",
						"test1":                 "test1",
						"test2":                 "test2",
						interpreter.BootTimeKey: bootTime1,
					},
				},
				interpreter.Event{
					Metadata: map[string]string{
						"test":                  "test",
						"test1":                 "test",
						"test3":                 "test3",
						interpreter.BootTimeKey: bootTime1,
					},
				},
				interpreter.Event{
					Metadata: map[string]string{
						"test":                  "test2",
						"test1":                 "test3",
						"test4":                 "test4",
						interpreter.BootTimeKey: bootTime2,
					},
				},
				interpreter.Event{
					Metadata: map[string]string{
						"test":                  "test3",
						"test1":                 "test3",
						interpreter.BootTimeKey: bootTime2,
					},
				},
			},
			expectedValid:         false,
			expectedInvalidFields: []string{"test", "test1"},
		},
		{
			description:   "empty list",
			fields:        []string{"test", "test1"},
			expectedValid: true,
			events:        []interpreter.Event{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			validator := MetadataValidator(tc.fields, tc.withinCycle)
			valid, err := validator.Valid(tc.events)
			assert.Equal(tc.expectedValid, valid)
			if tc.expectedValid {
				assert.Nil(err)
			} else {
				var cvErr CycleValidationErr
				assert.True(errors.As(err, &cvErr))
				assert.ElementsMatch(tc.expectedInvalidFields, cvErr.Fields())
			}
		})
	}
}

func TestSessionOnlineValidator(t *testing.T) {
	tests := []struct {
		description   string
		events        []interpreter.Event
		skipFunc      func(string) bool
		expectedValid bool
		expectedIDs   []string
	}{
		{
			description:   "empty list",
			events:        []interpreter.Event{},
			expectedValid: true,
		},
		{
			description: "all valid, no skip",
			skipFunc:    func(id string) bool { return false },
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "2",
				},
				interpreter.Event{
					Destination: "non-event",
					SessionID:   "5",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "3",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "3",
				},
			},
			expectedValid: true,
		},
		{
			description: "all valid, skip",
			skipFunc: func(id string) bool {
				return id == "3"
			},
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "2",
				},
				interpreter.Event{
					Destination: "non-event",
					SessionID:   "5",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "3",
				},
			},
			expectedValid: true,
		},
		{
			description: "invalid-no skip",
			skipFunc: func(id string) bool {
				return false
			},
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "2",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "3",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "4",
				},
				interpreter.Event{
					Destination: "non-event",
					SessionID:   "5",
				},
			},
			expectedValid: false,
			expectedIDs:   []string{"2", "3"},
		},
		{
			description: "invalid-skip",
			skipFunc: func(id string) bool {
				return id == "4"
			},
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "2",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "3",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "4",
				},
				interpreter.Event{
					Destination: "non-event",
					SessionID:   "5",
				},
			},
			expectedValid: false,
			expectedIDs:   []string{"2", "3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			validator := SessionOnlineValidator(tc.skipFunc)
			valid, err := validator.Valid(tc.events)
			assert.Equal(tc.expectedValid, valid)
			if !tc.expectedValid {
				var cvErr CycleValidationErr
				assert.True(errors.As(err, &cvErr))
				assert.ElementsMatch(tc.expectedIDs, cvErr.Fields())
				assert.Equal(validation.MissingOnlineEvent, cvErr.Tag())
			} else {
				assert.Nil(err)
			}
		})
	}
}

func TestSessionOfflineValidator(t *testing.T) {
	tests := []struct {
		description   string
		events        []interpreter.Event
		skipFunc      func(string) bool
		expectedValid bool
		expectedIDs   []string
	}{
		{
			description:   "empty list",
			events:        []interpreter.Event{},
			expectedValid: true,
		},
		{
			description: "invalid with skip",
			skipFunc: func(id string) bool {
				return id == "5"
			},
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/offline",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "2",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
				},
				interpreter.Event{
					Destination: "non-event",
					SessionID:   "3",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "5",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "4",
				},
			},
			expectedValid: false,
			expectedIDs:   []string{"2", "4"},
		},
		{
			description: "invalid without skip",
			skipFunc: func(id string) bool {
				return false
			},
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/offline",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "2",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
				},
				interpreter.Event{
					Destination: "non-event",
					SessionID:   "3",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "5",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "4",
				},
			},
			expectedValid: false,
			expectedIDs:   []string{"2", "4", "5"},
		},
		{
			description: "valid with skip",
			skipFunc: func(id string) bool {
				return id == "5"
			},
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/offline",
					SessionID:   "1",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "2",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/offline",
					SessionID:   "2",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
				},
				interpreter.Event{
					Destination: "non-event",
					SessionID:   "3",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					SessionID:   "5",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/some-event",
					SessionID:   "4",
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/offline",
					SessionID:   "4",
				},
			},
			expectedValid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			validator := SessionOfflineValidator(tc.skipFunc)
			valid, err := validator.Valid(tc.events)
			assert.Equal(tc.expectedValid, valid)
			if !tc.expectedValid {
				var cvErr CycleValidationErr
				assert.True(errors.As(err, &cvErr))
				assert.ElementsMatch(tc.expectedIDs, cvErr.Fields())
				assert.Equal(validation.MissingOfflineEvent, cvErr.Tag())
			} else {
				assert.Nil(err)
			}
		})
	}
}
func TestTransactionUUIDValidator(t *testing.T) {
	tests := []struct {
		description     string
		events          []interpreter.Event
		expectedValid   bool
		expectedRepeats []string
	}{
		{
			description: "valid",
			events: []interpreter.Event{
				interpreter.Event{TransactionUUID: "1"},
				interpreter.Event{TransactionUUID: "2"},
				interpreter.Event{TransactionUUID: "3"},
				interpreter.Event{TransactionUUID: "4"},
			},
			expectedValid: true,
		},
		{
			description: "invalid",
			events: []interpreter.Event{
				interpreter.Event{TransactionUUID: "1"},
				interpreter.Event{TransactionUUID: "2"},
				interpreter.Event{TransactionUUID: "1"},
				interpreter.Event{TransactionUUID: "4"},
				interpreter.Event{TransactionUUID: "3"},
				interpreter.Event{TransactionUUID: "3"},
			},
			expectedValid:   false,
			expectedRepeats: []string{"1", "3"},
		},
	}

	validator := TransactionUUIDValidator()
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			valid, err := validator.Valid(tc.events)
			assert.Equal(tc.expectedValid, valid)
			if tc.expectedValid {
				assert.Nil(err)
			} else {
				var cvErr CycleValidationErr
				assert.True(errors.As(err, &cvErr))
				assert.ElementsMatch(tc.expectedRepeats, cvErr.Fields())
			}
		})
	}
}
func TestDetermineMetadataValues(t *testing.T) {
	fields := []string{"test", "test1", "test2", "test3"}
	event := interpreter.Event{
		Metadata: map[string]string{
			"test1": "test1Val",
			"test2": "test2Val",
			"test0": "test0Val",
		},
	}

	expectedValues := map[string]string{
		"test":  "",
		"test1": "test1Val",
		"test2": "test2Val",
		"test3": "",
	}
	values := determineMetadataValues(fields, event)
	assert := assert.New(t)
	assert.Equal(len(expectedValues), len(values))
	for key, val := range expectedValues {
		v, found := values[key]
		assert.True(found)
		assert.Equal(val, v)
	}

}

func TestFindSessionsWithoutEvent(t *testing.T) {
	tests := []struct {
		description           string
		events                map[string]bool
		skipFunc              func(string) bool
		expectedInvalidFields []string
	}{
		{
			description: "valid without skip",
			events:      map[string]bool{"1": true, "2": true, "3": true},
		},
		{
			description: "valid with skip",
			events:      map[string]bool{"2": true, "3": true, "1": false},
			skipFunc: func(id string) bool {
				return id == "1"
			},
		},
		{
			description:           "invalid without skip",
			events:                map[string]bool{"2": true, "1": false, "3": false, "4": false},
			expectedInvalidFields: []string{"1", "3", "4"},
		},
		{
			description: "invalid with skip",
			events:      map[string]bool{"1": false, "2": true, "3": false, "4": false},
			skipFunc: func(id string) bool {
				return id == "1"
			},
			expectedInvalidFields: []string{"3", "4"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			invalidFields := findSessionsWithoutEvent(tc.events, tc.skipFunc)
			assert.Equal(len(tc.expectedInvalidFields), len(invalidFields))
			assert.ElementsMatch(tc.expectedInvalidFields, invalidFields)
		})
	}
}

func TestCheckMetadataValues(t *testing.T) {
	tests := []struct {
		description           string
		expectedMetadataVals  map[string]string
		incorrectMetadataVals map[string]bool
		event                 interpreter.Event
		expectedIncorrect     map[string]bool
	}{
		{
			description: "valid",
			expectedMetadataVals: map[string]string{
				"test1": "test1Val",
				"test2": "test2Val",
				"test3": "test3Val",
			},
			incorrectMetadataVals: make(map[string]bool),
			event: interpreter.Event{
				Metadata: map[string]string{
					"test1": "test1Val",
					"test2": "test2Val",
					"test3": "test3Val",
					"test4": "test4Val",
				},
			},
			expectedIncorrect: map[string]bool{},
		},
		{
			description: "invalid",
			expectedMetadataVals: map[string]string{
				"test1": "test1Val",
				"test2": "test2Val",
				"test3": "test3Val",
			},
			incorrectMetadataVals: make(map[string]bool),
			event: interpreter.Event{
				Metadata: map[string]string{
					"test1": "test1Val",
					"test2": "test",
					"test4": "test4Val",
				},
			},
			expectedIncorrect: map[string]bool{
				"test2": true,
				"test3": true,
			},
		},
		{
			description: "invalid with existing",
			expectedMetadataVals: map[string]string{
				"test1": "test1Val",
				"test2": "test2Val",
				"test3": "test3Val",
			},
			incorrectMetadataVals: map[string]bool{
				"test1": true,
			},
			event: interpreter.Event{
				Metadata: map[string]string{
					"test1": "test1Val",
					"test2": "test",
					"test4": "test4Val",
				},
			},
			expectedIncorrect: map[string]bool{
				"test2": true,
				"test3": true,
				"test1": true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			invalidFields := checkMetadataValues(tc.expectedMetadataVals, tc.incorrectMetadataVals, tc.event)
			assert.Equal(tc.expectedIncorrect, invalidFields)
		})
	}
}

func TestValidateMetadata(t *testing.T) {
	keys := []string{"test", "test0", "test1", "test2"}
	invalidEvents := []interpreter.Event{
		interpreter.Event{
			Metadata: map[string]string{
				"test":  "test",
				"test0": "test0",
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test":  "test1",
				"test0": "test0",
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test1": "test1",
				"test0": "test0",
			},
		},
	}

	validEvents := []interpreter.Event{
		interpreter.Event{
			Metadata: map[string]string{
				"test":  "test",
				"test0": "test0",
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test":  "test",
				"test0": "test0",
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test":  "test",
				"test0": "test0",
			},
		},
	}

	tests := []struct {
		description     string
		events          []interpreter.Event
		expectedInvalid []string
	}{
		{
			description:     "valid",
			events:          validEvents,
			expectedInvalid: nil,
		},
		{
			description:     "invalid",
			events:          invalidEvents,
			expectedInvalid: []string{"test", "test1"},
		},
		{
			description:     "empty",
			events:          []interpreter.Event{},
			expectedInvalid: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			invalidKeys := validateMetadata(keys, tc.events)
			assert.ElementsMatch(t, tc.expectedInvalid, invalidKeys)
		})
	}
}

func TestValidateMetadataWithinCycle(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)

	bootTime1 := fmt.Sprint(now.Unix())
	bootTime2 := fmt.Sprint(now.Add(time.Hour).Unix())
	invalidEvents := []interpreter.Event{
		interpreter.Event{
			Metadata: map[string]string{
				"test1":                 "test1val",
				"test2":                 "test2val",
				interpreter.BootTimeKey: bootTime1,
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test1":                 "test1val",
				"test2":                 "test",
				interpreter.BootTimeKey: bootTime1,
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test":                  "testval",
				"test0":                 "test0val",
				interpreter.BootTimeKey: bootTime2,
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test":                  "testval",
				"test0":                 "test0val",
				interpreter.BootTimeKey: bootTime2,
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test1": "test1random",
			},
		},
	}

	validEvents := []interpreter.Event{
		interpreter.Event{
			Metadata: map[string]string{
				"test1":                 "test1val",
				"test2":                 "test2val",
				interpreter.BootTimeKey: bootTime1,
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test1":                 "test1val",
				"test2":                 "test2val",
				interpreter.BootTimeKey: bootTime1,
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test1":                 "test1",
				"test2":                 "test2",
				interpreter.BootTimeKey: bootTime2,
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test1":                 "test1",
				"test2":                 "test2",
				interpreter.BootTimeKey: bootTime2,
			},
		},
		interpreter.Event{
			Metadata: map[string]string{
				"test1": "test1random",
			},
		},
	}

	fields := []string{"test1", "test2", "test0"}

	tests := []struct {
		description     string
		events          []interpreter.Event
		expectedInvalid []string
	}{
		{
			description:     "valid",
			events:          validEvents,
			expectedInvalid: nil,
		},
		{
			description:     "invalid",
			events:          invalidEvents,
			expectedInvalid: []string{"test2"},
		},
		{
			description:     "empty",
			events:          []interpreter.Event{},
			expectedInvalid: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			invalidKeys := validateMetadataWithinCycle(fields, tc.events)
			assert.ElementsMatch(t, tc.expectedInvalid, invalidKeys)
		})
	}
}
