package history

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

type testEvent struct {
	event interpreter.Event
	match bool
	valid bool
	err   error
}

func TestLastSessionFinder(t *testing.T) {
	t.Run("general errors", func(t *testing.T) { testError(t, true) })
	t.Run("duplicate and newer boot-time", func(t *testing.T) { testDuplicateAndNewer(t, true) })
	t.Run("event not found", func(t *testing.T) { testNotFound(t, true) })
	t.Run("success", func(t *testing.T) { testSuccess(t, true) })

}

func TestCurrentSessionFinder(t *testing.T) {
	t.Run("general errors", func(t *testing.T) { testError(t, false) })
	t.Run("duplicate and newer boot-time", func(t *testing.T) { testDuplicateAndNewer(t, false) })
	t.Run("event not found", func(t *testing.T) { testNotFound(t, false) })
	t.Run("success", func(t *testing.T) { testSuccess(t, false) })
}

func TestEventHistoryIterator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	fatalError := errors.New("invalid event")
	latestEvent := interpreter.Event{
		Destination:     "event:device-status/mac:112233445566/online",
		Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
		Birthdate:       now.UnixNano(),
		TransactionUUID: "latest",
	}

	tests := []struct {
		description   string
		events        []testEvent
		expectedEvent interpreter.Event
		latestEvent   interpreter.Event
		expectedErr   error
	}{
		{
			description: "valid",
			events: []testEvent{
				testEvent{
					event: latestEvent,
					match: false,
				},
				testEvent{
					event: interpreter.Event{
						Destination: "event:device-status/mac:112233445566/online",
						Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
					},
					match: false,
				},
			},
			latestEvent:   latestEvent,
			expectedEvent: latestEvent,
		},
		{
			description:   "no events",
			events:        []testEvent{},
			latestEvent:   latestEvent,
			expectedEvent: latestEvent,
		},
		{
			description: "same event",
			events: []testEvent{
				testEvent{
					event: latestEvent,
					match: false,
				},
			},
			latestEvent:   latestEvent,
			expectedEvent: latestEvent,
		},
		{
			description: "missing boot-time",
			events: []testEvent{
				testEvent{
					event: interpreter.Event{
						Destination: "event:device-status/mac:112233445566/online",
						Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
					},
					match: false,
				},
			},
			latestEvent: interpreter.Event{
				Destination: "event:device-status/mac:112233445566/online",
				Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
			},
			expectedEvent: interpreter.Event{},
			expectedErr:   validation.InvalidBootTimeErr{},
		},
		{
			description: "invalid boot-time",
			events: []testEvent{
				testEvent{
					event: interpreter.Event{
						Destination: "event:device-status/mac:112233445566/online",
						Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
					},
					match: false,
				},
			},
			latestEvent: interpreter.Event{
				Destination: "event:device-status/mac:112233445566/online",
				Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
				Metadata:    map[string]string{interpreter.BootTimeKey: "-1"},
			},
			expectedEvent: interpreter.Event{},
			expectedErr:   validation.InvalidBootTimeErr{},
		},
		{
			description: "invalid event",
			events: []testEvent{
				testEvent{
					event: interpreter.Event{
						Destination: "event:device-status/mac:112233445566/online",
						Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
					},
					match: false,
				},
				testEvent{
					event: interpreter.Event{
						Destination: "event:device-status/mac:112233445566/online",
						Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-3 * time.Minute).Unix())},
						Birthdate:   now.Add(-3 * time.Minute).UnixNano(),
					},
					match: true,
					err:   fatalError,
				},
			},
			latestEvent:   latestEvent,
			expectedEvent: interpreter.Event{},
			expectedErr:   fatalError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			comparator := new(mockComparator)
			events := make([]interpreter.Event, 0, len(tc.events))
			for _, te := range tc.events {
				comparator.On("Compare", te.event, tc.latestEvent).Return(te.match, te.err)
				events = append(events, te.event)
			}

			finder := EventHistoryIterator(comparator)
			event, err := finder.Find(events, tc.latestEvent)
			assert.Equal(tc.expectedEvent, event)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func testError(t *testing.T, past bool) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)

	comparator := new(mockComparator)
	fatalError := errors.New("invalid event")
	comparator.On("Compare", mock.Anything, mock.Anything).Return(true, fatalError)
	var finder FinderFunc
	if past {
		finder = LastSessionFinder(new(mockValidator), comparator)
	} else {
		finder = CurrentSessionFinder(new(mockValidator), comparator)
	}

	tests := []struct {
		description   string
		events        []interpreter.Event
		expectedEvent interpreter.Event
		latestEvent   interpreter.Event
		expectedErr   error
	}{
		{
			description: "Non-existent boot-time",
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
					Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
				},
			},
			latestEvent: interpreter.Event{
				Destination:     "event:device-status/mac:112233445566/online",
				Birthdate:       now.UnixNano(),
				TransactionUUID: "latest",
			},
			expectedEvent: interpreter.Event{},
			expectedErr:   validation.InvalidBootTimeErr{},
		},
		{
			description: "Invalid boot-time",
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
					Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
				},
			},
			latestEvent: interpreter.Event{
				Destination:     "event:device-status/mac:112233445566/online",
				Metadata:        map[string]string{interpreter.BootTimeKey: "-1"},
				Birthdate:       now.UnixNano(),
				TransactionUUID: "latest",
			},
			expectedEvent: interpreter.Event{},
			expectedErr:   validation.InvalidBootTimeErr{},
		},
		{
			description: "Fatal error",
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(1 * time.Hour).Unix())},
					Birthdate:   now.Add(1 * time.Hour).UnixNano(),
				},
			},
			latestEvent: interpreter.Event{
				Destination:     "event:device-status/mac:112233445566/online",
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate:       now.UnixNano(),
				TransactionUUID: "latest",
			},
			expectedEvent: interpreter.Event{},
			expectedErr:   fatalError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			event, err := finder.Find(tc.events, tc.latestEvent)
			assert.Equal(tc.expectedEvent, event)
			assert.NotNil(err)
			assert.Contains(err.Error(), tc.expectedErr.Error())
		})
	}
}

func testDuplicateAndNewer(t *testing.T, past bool) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	latestEvent := interpreter.Event{
		Destination:     "event:device-status/mac:112233445566/online",
		Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
		Birthdate:       now.UnixNano(),
		TransactionUUID: "latest",
	}

	comparator := Comparators([]Comparator{OlderBootTimeComparator(), DuplicateEventComparator()})
	var finder FinderFunc
	if past {
		finder = LastSessionFinder(new(mockValidator), comparator)
	} else {
		finder = CurrentSessionFinder(new(mockValidator), comparator)
	}

	tests := []struct {
		description   string
		events        []interpreter.Event
		expectedEvent interpreter.Event
		latestEvent   interpreter.Event
		expectedErr   error
	}{
		{
			description: "Newer boot-time found",
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(1 * time.Hour).Unix())},
					Birthdate:   now.Add(-1 * time.Hour).UnixNano(),
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
					Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
				},
			},
			latestEvent:   latestEvent,
			expectedEvent: interpreter.Event{},
			expectedErr:   errNewerBootTime,
		},
		{
			description: "Duplicate event found",
			events: []interpreter.Event{
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
					Birthdate:   now.UnixNano(),
				},
				interpreter.Event{
					Destination: "event:device-status/mac:112233445566/online",
					Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
					Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
				},
			},
			latestEvent:   latestEvent,
			expectedEvent: interpreter.Event{},
			expectedErr:   errDuplicateEvent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			tc.events = append(tc.events, tc.latestEvent)
			event, err := finder.Find(tc.events, tc.latestEvent)
			assert.Equal(tc.expectedEvent, event)
			assert.NotNil(err)
			assert.Contains(err.Error(), tc.expectedErr.Error())
		})
	}
}

func testNotFound(t *testing.T, past bool) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	latestEvent := interpreter.Event{
		Destination:     "event:device-status/mac:112233445566/online",
		Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
		Birthdate:       now.UnixNano(),
		TransactionUUID: "latest",
	}
	comparator := new(mockComparator)
	comparator.On("Compare", mock.Anything, mock.Anything).Return(false, nil)

	tests := []struct {
		description   string
		events        []testEvent
		expectedEvent interpreter.Event
		expectedErr   error
	}{
		{
			description:   "no events",
			events:        []testEvent{},
			expectedEvent: interpreter.Event{},
			expectedErr:   EventNotFoundErr,
		},
		{
			description: "same event",
			events: []testEvent{
				testEvent{
					event: latestEvent,
					valid: false,
				},
			},
			expectedEvent: interpreter.Event{},
			expectedErr:   EventNotFoundErr,
		},
		{
			description: "no events match",
			events: []testEvent{
				testEvent{
					event: interpreter.Event{
						Destination: "event:device-status/mac:112233445566/online",
						Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-1 * time.Hour).UnixNano(),
					},
					valid: false,
					err:   validation.ErrInvalidEventType,
				},
				testEvent{
					event: interpreter.Event{
						Destination: "event:device-status/mac:112233445566/online",
						Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
					},
					valid: false,
					err:   validation.ErrInvalidEventType,
				},
			},
			expectedEvent: interpreter.Event{},
			expectedErr:   EventNotFoundErr,
		},
		{
			description: "event found not from correct session",
			events: []testEvent{
				testEvent{
					event: interpreter.Event{
						Destination: "mac:112233445566/offline",
						Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-1 * time.Hour).UnixNano(),
					},
					valid: true,
				},
				testEvent{
					event: interpreter.Event{
						Destination: "event:device-status/mac:112233445566/online",
						Metadata:    map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-30 * time.Minute).Unix())},
						Birthdate:   now.Add(-1 * time.Hour).UnixNano(),
					},
					valid: false,
					err:   validation.ErrInvalidEventType,
				},
			},
			expectedEvent: interpreter.Event{},
			expectedErr:   EventNotFoundErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			mockVal := new(mockValidator)
			testEvents := make([]interpreter.Event, 0, len(tc.events))
			for _, te := range tc.events {
				mockVal.On("Valid", te.event).Return(te.valid, te.err)
				testEvents = append(testEvents, te.event)
			}
			var finder FinderFunc
			if past {
				finder = LastSessionFinder(mockVal, comparator)
			} else {
				finder = CurrentSessionFinder(mockVal, comparator)
			}
			event, err := finder(testEvents, latestEvent)
			assert.Equal(tc.expectedEvent, event)
			assert.NotNil(err)
			assert.True(errors.Is(err, tc.expectedErr),
				fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
					err, tc.expectedErr),
			)
		})
	}
}

func testSuccess(t *testing.T, past bool) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	mockVal := new(mockValidator)
	comparator := new(mockComparator)
	comparator.On("Compare", mock.Anything, mock.Anything).Return(false, nil)

	latestEvent := interpreter.Event{
		Destination:     "event:device-status/mac:112233445566/online",
		Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
		Birthdate:       now.UnixNano(),
		TransactionUUID: "latest",
	}
	var validEvent interpreter.Event
	if past {
		validEvent = interpreter.Event{
			Destination:     "event:device-status/mac:112233445566/online",
			Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
			Birthdate:       now.Add(-1 * time.Hour).UnixNano(),
			TransactionUUID: "test",
		}
	} else {
		validEvent = interpreter.Event{
			Destination:     "event:device-status/mac:112233445566/online",
			Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
			Birthdate:       now.UnixNano(),
			TransactionUUID: "test",
		}
	}

	testEvents := []testEvent{
		testEvent{
			event: interpreter.Event{
				Destination:     "event:device-status/mac:112233445566/online",
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-2 * time.Hour).Unix())},
				Birthdate:       now.Add(-2 * time.Hour).UnixNano(),
				TransactionUUID: "test",
			},
			valid: true,
		},
		testEvent{
			event: validEvent,
			valid: true,
		},
		testEvent{
			event: interpreter.Event{
				Destination:     "event:device-status/mac:112233445566/online",
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
				Birthdate:       now.Add(-30 * time.Minute).UnixNano(),
				TransactionUUID: "test",
			},
			valid: true,
		},
		testEvent{
			event: interpreter.Event{
				Destination:     "event:device-status/mac:112233445566/online",
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate:       now.Add(time.Minute).UnixNano(),
				TransactionUUID: "test",
			},
			valid: true,
		},
		testEvent{
			event: interpreter.Event{
				Destination:     "mac:112233445566/offline",
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
				Birthdate:       now.Add(-1 * time.Hour).UnixNano(),
				TransactionUUID: "test",
			},
			valid: false,
			err:   validation.ErrInvalidEventType,
		},
		testEvent{
			event: interpreter.Event{
				Destination:     "event:device-status/mac:112233445566/online",
				Birthdate:       now.Add(-1 * time.Hour).UnixNano(),
				TransactionUUID: "test",
			},
			valid: true,
		},
	}

	events := make([]interpreter.Event, 0, len(testEvents)+1)
	events = append(events, latestEvent)
	for _, te := range testEvents {
		mockVal.On("Valid", te.event).Return(te.valid, te.err)
		events = append(events, te.event)
	}

	assert := assert.New(t)
	var finder FinderFunc
	if past {
		finder = LastSessionFinder(mockVal, comparator)
	} else {
		finder = CurrentSessionFinder(mockVal, comparator)
	}
	event, err := finder.Find(events, latestEvent)
	assert.Equal(validEvent, event)
	assert.Nil(err)

}

func TestGetPreviousBootTime(t *testing.T) {
	tests := []struct {
		description    string
		currentTime    int64
		defaultTime    int64
		latestBootTime int64
		expectedTime   int64
		expectedNew    bool
	}{
		{
			description:    "New boot-time returned",
			currentTime:    60,
			defaultTime:    50,
			latestBootTime: 70,
			expectedTime:   60,
			expectedNew:    true,
		},
		{
			description:    "Default boot-time returned",
			currentTime:    40,
			defaultTime:    50,
			latestBootTime: 70,
			expectedTime:   50,
		},
		{
			description:    "New boot-time > latest",
			currentTime:    80,
			defaultTime:    50,
			latestBootTime: 70,
			expectedTime:   50,
		},
		{
			description:    "New boot-time = latest",
			currentTime:    70,
			defaultTime:    50,
			latestBootTime: 70,
			expectedTime:   50,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			event := interpreter.Event{
				Metadata: map[string]string{interpreter.BootTimeKey: fmt.Sprint(tc.currentTime)},
			}
			bootTime, newFound := getPreviousBootTime(event, tc.defaultTime, tc.latestBootTime)
			assert.Equal(tc.expectedTime, bootTime)
			assert.Equal(tc.expectedNew, newFound)
		})
	}
}

func TestNewValidEvent(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	tests := []struct {
		description    string
		newEvent       interpreter.Event
		defaultEvent   interpreter.Event
		newEventValid  bool
		targetBootTime int64
		expectedRes    bool
	}{
		{
			description: "new event returned",
			newEvent: interpreter.Event{
				Metadata:  map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate: now.UnixNano(),
			},
			defaultEvent: interpreter.Event{
				Metadata:  map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(time.Hour).Unix())},
				Birthdate: now.Add(time.Hour).UnixNano(),
			},
			newEventValid:  true,
			targetBootTime: now.Unix(),
			expectedRes:    true,
		},
		{
			description: "target boot-time mismatch",
			newEvent: interpreter.Event{
				Metadata:  map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(2 * time.Hour).Unix())},
				Birthdate: now.Add(2 * time.Hour).UnixNano(),
			},
			defaultEvent: interpreter.Event{
				Metadata:  map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(time.Hour).Unix())},
				Birthdate: now.Add(time.Hour).UnixNano(),
			},
			newEventValid:  true,
			targetBootTime: now.Unix(),
			expectedRes:    false,
		},
		{
			description: "event invalid",
			newEvent: interpreter.Event{
				Metadata:  map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate: now.UnixNano(),
			},
			defaultEvent: interpreter.Event{
				Metadata:  map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(time.Hour).Unix())},
				Birthdate: now.Add(time.Hour).UnixNano(),
			},
			newEventValid:  false,
			targetBootTime: now.Unix(),
			expectedRes:    false,
		},
		{
			description: "both event boot-times match",
			newEvent: interpreter.Event{
				Metadata:  map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate: now.Add(time.Minute).UnixNano(),
			},
			defaultEvent: interpreter.Event{
				Metadata:  map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate: now.UnixNano(),
			},
			newEventValid:  true,
			targetBootTime: now.Unix(),
			expectedRes:    false,
		},
		{
			description: "both event boot-times match",
			newEvent: interpreter.Event{
				Metadata:  map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate: now.UnixNano(),
			},
			defaultEvent: interpreter.Event{
				Metadata:  map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate: now.Add(time.Minute).UnixNano(),
			},
			newEventValid:  true,
			targetBootTime: now.Unix(),
			expectedRes:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			val := new(mockValidator)
			val.On("Valid", tc.newEvent).Return(tc.newEventValid, nil)
			newEventFound := newEventValid(tc.newEvent, tc.defaultEvent, val, tc.targetBootTime)
			assert.Equal(tc.expectedRes, newEventFound)
		})
	}
}
