package history

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

func TestComparators(t *testing.T) {
	assert := assert.New(t)
	testEvent := interpreter.Event{}
	comparators := Comparators([]Comparator{testComparator(false, nil), testComparator(false, nil)})
	match, err := comparators.Compare(testEvent, interpreter.Event{})
	assert.False(match)
	assert.Nil(err)

	comparators = Comparators([]Comparator{
		testComparator(false, nil),
		testComparator(true, errors.New("invalid event")),
		testComparator(true, errors.New("another invalid event")),
	})
	match, err = comparators.Compare(testEvent, interpreter.Event{})
	assert.True(match)
	assert.Equal(errors.New("invalid event"), err)
}

func testComparator(returnBool bool, returnErr error) ComparatorFunc {
	return func(compareTo interpreter.Event, comparedEvent interpreter.Event) (bool, error) {
		return returnBool, returnErr
	}
}

func TestOlderBootTimeComparator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	latestEvent := interpreter.Event{
		Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
		TransactionUUID: "123",
	}

	comparator := OlderBootTimeComparator()
	tests := []struct {
		description   string
		historyEvent  interpreter.Event
		incomingEvent interpreter.Event
		match         bool
		expectedErr   error
	}{
		{
			description: "valid event",
			historyEvent: interpreter.Event{
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(-30 * time.Minute).Unix()),
				},
				TransactionUUID: "abc",
			},
			incomingEvent: latestEvent,
			match:         false,
		},
		{
			description: "same event uuid",
			historyEvent: interpreter.Event{
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(30 * time.Minute).Unix()),
				},
				TransactionUUID: "123",
			},
			incomingEvent: latestEvent,
			match:         false,
		},
		{
			description: "boot-time not present",
			historyEvent: interpreter.Event{
				Metadata:        map[string]string{},
				TransactionUUID: "abc",
			},
			match: false,
		},
		{
			description: "newer boot-time",
			historyEvent: interpreter.Event{
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(30 * time.Minute).Unix())},
				TransactionUUID: "abc",
			},
			incomingEvent: latestEvent,
			match:         true,
			expectedErr:   errNewerBootTime,
		},
		{
			description: "latest boot-time invalid",
			historyEvent: interpreter.Event{
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-30 * time.Minute).Unix())},
				TransactionUUID: "abc",
			},
			incomingEvent: interpreter.Event{
				TransactionUUID: "123",
			},
			match:       true,
			expectedErr: errNewerBootTime,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			match, err := comparator.Compare(tc.historyEvent, tc.incomingEvent)
			assert.Equal(tc.match, match)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.True(errors.Is(err, tc.expectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedErr),
				)

				var logError validation.MetricsLogError
				assert.True(errors.As(err, &logError))
				assert.Equal(newerBootTimeReason, logError.ErrorLabel())
			}
		})
	}
}

func TestDuplicateEventComparator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	destRegex := regexp.MustCompile(".*/online")
	latestEvent := interpreter.Event{
		Destination:     "mac:112233445566/online",
		Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
		TransactionUUID: "test",
		Birthdate:       now.UnixNano(),
	}

	comparator := DuplicateEventComparator(destRegex)
	tests := []struct {
		description   string
		historyEvent  interpreter.Event
		incomingEvent interpreter.Event
		match         bool
		expectedErr   error
	}{
		{
			description: "valid event",
			historyEvent: interpreter.Event{
				Destination: "mac:112233445566/online",
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(-30 * time.Minute).Unix()),
				},
				TransactionUUID: "abc",
				Birthdate:       now.Add(-30 * time.Minute).UnixNano(),
			},
			incomingEvent: latestEvent,
			match:         false,
		},
		{
			description: "same event uuid",
			historyEvent: interpreter.Event{
				Destination: "mac:112233445566/online",
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
				TransactionUUID: "test",
				Birthdate:       now.UnixNano(),
			},
			incomingEvent: latestEvent,
			match:         false,
		},
		{
			description: "event type mismatch",
			historyEvent: interpreter.Event{
				Destination: "mac:112233445566/offline",
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
				TransactionUUID: "abc",
				Birthdate:       now.UnixNano(),
			},
			incomingEvent: latestEvent,
			match:         false,
		},
		{
			description: "new event type mismatch",
			historyEvent: interpreter.Event{
				Destination: "mac:112233445566/online",
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
				TransactionUUID: "abc",
				Birthdate:       now.UnixNano(),
			},
			incomingEvent: interpreter.Event{
				Destination: "mac:112233445566/offline",
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
				TransactionUUID: "123",
				Birthdate:       now.Add(time.Minute).UnixNano(),
			},
			match: false,
		},
		{
			description: "boot-time missing",
			historyEvent: interpreter.Event{
				Destination:     "mac:112233445566/online",
				Metadata:        map[string]string{},
				TransactionUUID: "abc",
				Birthdate:       now.UnixNano(),
			},
			incomingEvent: latestEvent,
			match:         false,
		},
		{
			description: "duplicate found, older birthdate",
			historyEvent: interpreter.Event{
				Destination:     "mac:112233445566/online",
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				TransactionUUID: "abc",
				Birthdate:       now.Add(-1 * time.Minute).UnixNano(),
			},
			incomingEvent: latestEvent,
			match:         true,
			expectedErr:   errDuplicateEvent,
		},
		{
			description: "duplicate found, same birthdate",
			historyEvent: interpreter.Event{
				Destination:     "mac:112233445566/online",
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				TransactionUUID: "abc",
				Birthdate:       now.UnixNano(),
			},
			incomingEvent: latestEvent,
			match:         true,
			expectedErr:   errDuplicateEvent,
		},
		{
			description: "duplicate found, later birthdate",
			historyEvent: interpreter.Event{
				Destination:     "mac:112233445566/online",
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				TransactionUUID: "abc",
				Birthdate:       now.Add(time.Minute).UnixNano(),
			},
			incomingEvent: latestEvent,
			match:         false,
		},
		{
			description: "latest boot-time invalid",
			historyEvent: interpreter.Event{
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				TransactionUUID: "123",
				Birthdate:       now.UnixNano(),
			},
			incomingEvent: interpreter.Event{
				TransactionUUID: "test",
				Birthdate:       now.UnixNano(),
			},
			match: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			match, err := comparator.Compare(tc.historyEvent, tc.incomingEvent)
			assert.Equal(tc.match, match)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.True(errors.Is(err, tc.expectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedErr),
				)
				var logError validation.MetricsLogError
				assert.True(errors.As(err, &logError))
				assert.Equal(duplicateEventReason, logError.ErrorLabel())
			}
		})
	}
}
