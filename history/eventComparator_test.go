package history

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/interpreter"
)

func TestComparators(t *testing.T) {
	assert := assert.New(t)
	testEvent := interpreter.Event{}
	comparators := Comparators([]Comparator{testComparator(true, nil), testComparator(true, nil)})
	valid, err := comparators.Compare(testEvent, interpreter.Event{})
	assert.True(valid)
	assert.Nil(err)

	comparators = Comparators([]Comparator{
		testComparator(true, nil),
		testComparator(false, errors.New("invalid event")),
		testComparator(false, errors.New("another invalid event")),
	})
	valid, err = comparators.Compare(testEvent, interpreter.Event{})
	assert.False(valid)
	assert.Equal(errors.New("invalid event"), err)
}

func testComparator(returnBool bool, returnErr error) ComparatorFunc {
	return func(compareTo interpreter.Event, comparedEvent interpreter.Event) (bool, error) {
		return returnBool, returnErr
	}
}

func TestNewestBootTimeComparator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	latestEvent := interpreter.Event{
		Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
		TransactionUUID: "123",
	}

	comparator := NewestBootTimeComparator()
	tests := []struct {
		description   string
		historyEvent  interpreter.Event
		incomingEvent interpreter.Event
		valid         bool
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
			valid:         true,
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
			valid:         true,
		},
		{
			description: "boot-time not present",
			historyEvent: interpreter.Event{
				Metadata:        map[string]string{},
				TransactionUUID: "abc",
			},
			valid: true,
		},
		{
			description: "newer boot-time",
			historyEvent: interpreter.Event{
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(30 * time.Minute).Unix())},
				TransactionUUID: "abc",
			},
			incomingEvent: latestEvent,
			valid:         false,
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
			valid:       false,
			expectedErr: errNewerBootTime,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			valid, err := comparator.Compare(tc.historyEvent, tc.incomingEvent)
			assert.Equal(tc.valid, valid)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.True(errors.Is(err, tc.expectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedErr),
				)
			}
		})
	}
}

func TestUniqueEventComparator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	destRegex := regexp.MustCompile(".*/online")
	latestEvent := interpreter.Event{
		Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
		TransactionUUID: "test",
		Birthdate:       now.UnixNano(),
	}

	comparator := UniqueEventComparator(destRegex)
	tests := []struct {
		description   string
		historyEvent  interpreter.Event
		incomingEvent interpreter.Event
		valid         bool
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
			valid:         true,
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
			valid:         true,
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
			valid:         true,
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
			valid:         true,
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
			valid:         false,
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
			valid:         false,
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
			valid:         true,
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
			valid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			valid, err := comparator.Compare(tc.historyEvent, tc.incomingEvent)
			assert.Equal(tc.valid, valid)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.True(errors.Is(err, tc.expectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedErr),
				)
			}
		})
	}
}
