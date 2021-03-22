package history

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/interpreter/message"
	"github.com/xmidt-org/interpreter/validation"
)

type testEvent struct {
	event message.Event
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

func TestSameEventFinder(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	fatalError := errors.New("invalid event")
	latestEvent := message.Event{
		Destination:     "mac:112233445566/online",
		Metadata:        map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
		Birthdate:       now.UnixNano(),
		TransactionUUID: "latest",
	}

	tests := []struct {
		description   string
		events        []testEvent
		expectedEvent message.Event
		latestEvent   message.Event
		expectedErr   error
	}{
		{
			description: "valid",
			events: []testEvent{
				testEvent{
					event: latestEvent,
					valid: true,
				},
				testEvent{
					event: message.Event{
						Destination: "mac:112233445566/online",
						Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
					},
					valid: true,
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
					valid: true,
				},
			},
			latestEvent:   latestEvent,
			expectedEvent: latestEvent,
		},
		{
			description: "missing boot-time",
			events: []testEvent{
				testEvent{
					event: message.Event{
						Destination: "mac:112233445566/online",
						Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
					},
					valid: true,
				},
			},
			latestEvent: message.Event{
				Destination: "mac:112233445566/online",
				Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
			},
			expectedEvent: message.Event{},
			expectedErr:   validation.InvalidBootTimeErr{},
		},
		{
			description: "invalid boot-time",
			events: []testEvent{
				testEvent{
					event: message.Event{
						Destination: "mac:112233445566/online",
						Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
					},
					valid: true,
				},
			},
			latestEvent: message.Event{
				Destination: "mac:112233445566/online",
				Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
				Metadata:    map[string]string{message.BootTimeKey: "-1"},
			},
			expectedEvent: message.Event{},
			expectedErr:   validation.InvalidBootTimeErr{},
		},
		{
			description: "invalid event",
			events: []testEvent{
				testEvent{
					event: message.Event{
						Destination: "mac:112233445566/online",
						Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
					},
					valid: true,
				},
				testEvent{
					event: message.Event{
						Destination: "mac:112233445566/online",
						Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-3 * time.Minute).Unix())},
						Birthdate:   now.Add(-3 * time.Minute).UnixNano(),
					},
					valid: false,
					err:   fatalError,
				},
			},
			latestEvent: latestEvent,
			expectedEvent: message.Event{
				Destination: "mac:112233445566/online",
				Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-3 * time.Minute).Unix())},
				Birthdate:   now.Add(-3 * time.Minute).UnixNano(),
			},
			expectedErr: fatalError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			fatalValidators := new(mockValidator)
			events := make([]message.Event, 0, len(tc.events))
			for _, te := range tc.events {
				fatalValidators.On("Valid", te.event).Return(te.valid, te.err)
				events = append(events, te.event)
			}

			finder := EventHistoryIterator(fatalValidators)
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

	fatalValidators := new(mockValidator)
	fatalError := errors.New("invalid event")
	fatalValidators.On("Valid", mock.Anything).Return(false, fatalError)
	var finder FinderFunc
	if past {
		finder = LastSessionFinder(new(mockValidator), fatalValidators)
	} else {
		finder = CurrentSessionFinder(new(mockValidator), fatalValidators)
	}

	tests := []struct {
		description   string
		events        []message.Event
		expectedEvent message.Event
		latestEvent   message.Event
		expectedErr   error
	}{
		{
			description: "Non-existent boot-time",
			events: []message.Event{
				message.Event{
					Destination: "mac:112233445566/online",
					Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
					Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
				},
			},
			latestEvent: message.Event{
				Destination:     "mac:112233445566/online",
				Birthdate:       now.UnixNano(),
				TransactionUUID: "latest",
			},
			expectedEvent: message.Event{},
			expectedErr:   validation.InvalidBootTimeErr{},
		},
		{
			description: "Invalid boot-time",
			events: []message.Event{
				message.Event{
					Destination: "mac:112233445566/online",
					Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
					Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
				},
			},
			latestEvent: message.Event{
				Destination:     "mac:112233445566/online",
				Metadata:        map[string]string{message.BootTimeKey: "-1"},
				Birthdate:       now.UnixNano(),
				TransactionUUID: "latest",
			},
			expectedEvent: message.Event{},
			expectedErr:   validation.InvalidBootTimeErr{},
		},
		{
			description: "Fatal error",
			events: []message.Event{
				message.Event{
					Destination: "mac:112233445566/online",
					Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(1 * time.Hour).Unix())},
					Birthdate:   now.Add(1 * time.Hour).UnixNano(),
				},
			},
			latestEvent: message.Event{
				Destination:     "mac:112233445566/online",
				Metadata:        map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate:       now.UnixNano(),
				TransactionUUID: "latest",
			},
			expectedEvent: message.Event{},
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
	regex := regexp.MustCompile(".*/online")
	latestEvent := message.Event{
		Destination:     "mac:112233445566/online",
		Metadata:        map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
		Birthdate:       now.UnixNano(),
		TransactionUUID: "latest",
	}

	fatalValidators := validation.Validators([]validation.Validator{
		validation.NewestBootTimeValidator(latestEvent), validation.UniqueEventValidator(latestEvent, regex),
	})
	var finder FinderFunc
	if past {
		finder = LastSessionFinder(new(mockValidator), fatalValidators)
	} else {
		finder = CurrentSessionFinder(new(mockValidator), fatalValidators)
	}

	tests := []struct {
		description   string
		events        []message.Event
		expectedEvent message.Event
		latestEvent   message.Event
		expectedErr   error
	}{
		{
			description: "Newer boot-time found",
			events: []message.Event{
				message.Event{
					Destination: "mac:112233445566/online",
					Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(1 * time.Hour).Unix())},
					Birthdate:   now.Add(-1 * time.Hour).UnixNano(),
				},
				message.Event{
					Destination: "mac:112233445566/online",
					Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
					Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
				},
			},
			latestEvent:   latestEvent,
			expectedEvent: message.Event{},
			expectedErr:   validation.InvalidEventErr{},
		},
		{
			description: "Duplicate event found",
			events: []message.Event{
				message.Event{
					Destination: "mac:112233445566/online",
					Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
					Birthdate:   now.UnixNano(),
				},
				message.Event{
					Destination: "mac:112233445566/online",
					Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
					Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
				},
			},
			latestEvent:   latestEvent,
			expectedEvent: message.Event{},
			expectedErr:   validation.InvalidEventErr{},
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
	latestEvent := message.Event{
		Destination:     "mac:112233445566/online",
		Metadata:        map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
		Birthdate:       now.UnixNano(),
		TransactionUUID: "latest",
	}
	fatalValidators := new(mockValidator)
	fatalValidators.On("Valid", mock.Anything).Return(true, nil)

	tests := []struct {
		description   string
		events        []testEvent
		expectedEvent message.Event
		expectedErr   error
	}{
		{
			description:   "no events",
			events:        []testEvent{},
			expectedEvent: message.Event{},
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
			expectedEvent: message.Event{},
			expectedErr:   EventNotFoundErr,
		},
		{
			description: "no events match",
			events: []testEvent{
				testEvent{
					event: message.Event{
						Destination: "mac:112233445566/online",
						Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-1 * time.Hour).UnixNano(),
					},
					valid: false,
					err:   validation.ErrInvalidEventType,
				},
				testEvent{
					event: message.Event{
						Destination: "mac:112233445566/online",
						Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-30 * time.Minute).UnixNano(),
					},
					valid: false,
					err:   validation.ErrInvalidEventType,
				},
			},
			expectedEvent: message.Event{},
			expectedErr:   EventNotFoundErr,
		},
		{
			description: "event matched not from correct session",
			events: []testEvent{
				testEvent{
					event: message.Event{
						Destination: "mac:112233445566/offline",
						Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
						Birthdate:   now.Add(-1 * time.Hour).UnixNano(),
					},
					valid: true,
				},
				testEvent{
					event: message.Event{
						Destination: "mac:112233445566/online",
						Metadata:    map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-30 * time.Minute).Unix())},
						Birthdate:   now.Add(-1 * time.Hour).UnixNano(),
					},
					valid: false,
					err:   validation.ErrInvalidEventType,
				},
			},
			expectedEvent: message.Event{},
			expectedErr:   EventNotFoundErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			mockVal := new(mockValidator)
			testEvents := make([]message.Event, 0, len(tc.events))
			for _, te := range tc.events {
				mockVal.On("Valid", te.event).Return(te.valid, te.err)
				testEvents = append(testEvents, te.event)
			}
			var finder FinderFunc
			if past {
				finder = LastSessionFinder(mockVal, fatalValidators)
			} else {
				finder = CurrentSessionFinder(mockVal, fatalValidators)
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
	fatalValidators := new(mockValidator)
	fatalValidators.On("Valid", mock.Anything).Return(true, nil)

	latestEvent := message.Event{
		Destination:     "mac:112233445566/online",
		Metadata:        map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
		Birthdate:       now.UnixNano(),
		TransactionUUID: "latest",
	}
	var validEvent message.Event
	if past {
		validEvent = message.Event{
			Destination:     "mac:112233445566/online",
			Metadata:        map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
			Birthdate:       now.Add(-1 * time.Hour).UnixNano(),
			TransactionUUID: "test",
		}
	} else {
		validEvent = message.Event{
			Destination:     "mac:112233445566/online",
			Metadata:        map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
			Birthdate:       now.UnixNano(),
			TransactionUUID: "test",
		}
	}

	testEvents := []testEvent{
		testEvent{
			event: message.Event{
				Destination:     "mac:112233445566/online",
				Metadata:        map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-2 * time.Hour).Unix())},
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
			event: message.Event{
				Destination:     "mac:112233445566/offline",
				Metadata:        map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix())},
				Birthdate:       now.Add(-1 * time.Hour).UnixNano(),
				TransactionUUID: "test",
			},
			valid: false,
			err:   validation.ErrInvalidEventType,
		},
		testEvent{
			event: message.Event{
				Destination:     "mac:112233445566/online",
				Birthdate:       now.Add(-1 * time.Hour).UnixNano(),
				TransactionUUID: "test",
			},
			valid: true,
		},
	}

	events := make([]message.Event, 0, len(testEvents)+1)
	events = append(events, latestEvent)
	for _, te := range testEvents {
		mockVal.On("Valid", te.event).Return(te.valid, te.err)
		events = append(events, te.event)
	}

	assert := assert.New(t)
	var finder FinderFunc
	if past {
		finder = LastSessionFinder(mockVal, fatalValidators)
	} else {
		finder = CurrentSessionFinder(mockVal, fatalValidators)
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
			event := message.Event{
				Metadata: map[string]string{message.BootTimeKey: fmt.Sprint(tc.currentTime)},
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
		newEvent       message.Event
		defaultEvent   message.Event
		newEventValid  bool
		targetBootTime int64
		expectedRes    bool
	}{
		{
			description: "new event returned",
			newEvent: message.Event{
				Metadata:  map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate: now.UnixNano(),
			},
			defaultEvent: message.Event{
				Metadata:  map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(time.Hour).Unix())},
				Birthdate: now.Add(time.Hour).UnixNano(),
			},
			newEventValid:  true,
			targetBootTime: now.Unix(),
			expectedRes:    true,
		},
		{
			description: "target boot-time mismatch",
			newEvent: message.Event{
				Metadata:  map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(2 * time.Hour).Unix())},
				Birthdate: now.Add(2 * time.Hour).UnixNano(),
			},
			defaultEvent: message.Event{
				Metadata:  map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(time.Hour).Unix())},
				Birthdate: now.Add(time.Hour).UnixNano(),
			},
			newEventValid:  true,
			targetBootTime: now.Unix(),
			expectedRes:    false,
		},
		{
			description: "event invalid",
			newEvent: message.Event{
				Metadata:  map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate: now.UnixNano(),
			},
			defaultEvent: message.Event{
				Metadata:  map[string]string{message.BootTimeKey: fmt.Sprint(now.Add(time.Hour).Unix())},
				Birthdate: now.Add(time.Hour).UnixNano(),
			},
			newEventValid:  false,
			targetBootTime: now.Unix(),
			expectedRes:    false,
		},
		{
			description: "both event boot-times match",
			newEvent: message.Event{
				Metadata:  map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate: now.Add(time.Minute).UnixNano(),
			},
			defaultEvent: message.Event{
				Metadata:  map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate: now.UnixNano(),
			},
			newEventValid:  true,
			targetBootTime: now.Unix(),
			expectedRes:    false,
		},
		{
			description: "both event boot-times match",
			newEvent: message.Event{
				Metadata:  map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
				Birthdate: now.UnixNano(),
			},
			defaultEvent: message.Event{
				Metadata:  map[string]string{message.BootTimeKey: fmt.Sprint(now.Unix())},
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
