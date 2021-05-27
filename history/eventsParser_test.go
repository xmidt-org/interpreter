package history

import (
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/xmidt-org/interpreter/validation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/interpreter"
)

type CycleTestSuite struct {
	suite.Suite
	Events []interpreter.Event
}

type testEventSetup struct {
	bootTime  time.Time
	numEvents int
}

func (suite *CycleTestSuite) createEvents(eventSetups ...testEventSetup) {
	var events []interpreter.Event
	for _, setup := range eventSetups {
		for i := 0; i < setup.numEvents; i++ {
			event := interpreter.Event{
				TransactionUUID: fmt.Sprintf("%d-%d", setup.bootTime.Unix(), i+1),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(setup.bootTime.Unix()),
				},
				Birthdate: setup.bootTime.Add(time.Duration(i) * time.Minute).UnixNano(),
			}
			events = append(events, event)
		}
	}

	events = append(events, interpreter.Event{TransactionUUID: "no-boottime"})
	suite.Events = events
}

func (suite *CycleTestSuite) parseEvents(from interpreter.Event, to interpreter.Event) []interpreter.Event {
	eventsCopy := suite.Events
	sort.Slice(eventsCopy, func(a, b int) bool {
		boottimeA, _ := eventsCopy[a].BootTime()
		boottimeB, _ := eventsCopy[b].BootTime()
		if boottimeA != boottimeB {
			return boottimeA < boottimeB
		}

		return eventsCopy[a].Birthdate < eventsCopy[b].Birthdate

	})

	fromIndex := 0
	toIndex := len(eventsCopy) - 1
	for i, event := range eventsCopy {
		if event.TransactionUUID == from.TransactionUUID {
			fromIndex = i
		} else if event.TransactionUUID == to.TransactionUUID {
			toIndex = i
		}
	}

	if fromIndex > toIndex {
		return eventsCopy
	}

	return eventsCopy[fromIndex : toIndex+1]
}

func (suite *CycleTestSuite) setEventDestination(eventID string, destination string) interpreter.Event {
	for i, event := range suite.Events {
		if event.TransactionUUID == eventID {
			event.Destination = destination
			suite.Events[i] = event
			return event
		}
	}

	return interpreter.Event{}
}

func (suite *CycleTestSuite) TestValid() {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	suite.Nil(err)

	futureBootTime := now.Add(1 * time.Hour)
	currentBootTime := now
	prevBootTime := now.Add(-1 * time.Hour)
	olderBootTime := now.Add(-2 * time.Hour)
	bootTimes := []testEventSetup{
		testEventSetup{
			bootTime:  currentBootTime,
			numEvents: 3,
		},
		testEventSetup{
			bootTime:  olderBootTime,
			numEvents: 3,
		},
		testEventSetup{
			bootTime:  prevBootTime,
			numEvents: 4,
		},
		testEventSetup{
			bootTime:  futureBootTime,
			numEvents: 2,
		},
	}

	suite.createEvents(bootTimes...)
	mockVal := new(mockValidator)
	mockComparator := new(mockComparator)
	mockVal.On("Valid", mock.Anything).Return(true, nil)
	mockComparator.On("Compare", mock.Anything, mock.Anything).Return(false, nil)
	fromEvent := suite.setEventDestination(fmt.Sprintf("%d-%d", prevBootTime.Unix(), 2), "event:device-status/mac:112233445566/reboot-pending")
	suite.setEventDestination(fmt.Sprintf("%d-%d", prevBootTime.Unix(), 3), "event:device-status/mac:112233445566/offline")
	toEvent := suite.setEventDestination(fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2), "event-device-status/mac:112233445566/some-event")
	expectedEvents := suite.parseEvents(fromEvent, toEvent)
	parser := BootCycleParser(mockComparator, mockVal)
	results, err := parser.Parse(suite.Events, toEvent)
	suite.Equal(expectedEvents, results)
	suite.Nil(err)
}

func (suite *CycleTestSuite) TestNoRebootPendingEvent() {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	suite.Nil(err)

	futureBootTime := now.Add(1 * time.Hour)
	currentBootTime := now
	prevBootTime := now.Add(-1 * time.Hour)
	olderBootTime := now.Add(-2 * time.Hour)
	bootTimes := []testEventSetup{
		testEventSetup{
			bootTime:  currentBootTime,
			numEvents: 3,
		},
		testEventSetup{
			bootTime:  olderBootTime,
			numEvents: 3,
		},
		testEventSetup{
			bootTime:  prevBootTime,
			numEvents: 4,
		},
		testEventSetup{
			bootTime:  futureBootTime,
			numEvents: 2,
		},
	}

	suite.createEvents(bootTimes...)
	mockVal := new(mockValidator)
	mockComparator := new(mockComparator)
	mockVal.On("Valid", mock.Anything).Return(true, nil)
	mockComparator.On("Compare", mock.Anything, mock.Anything).Return(false, nil)
	suite.setEventDestination(fmt.Sprintf("%d-%d", prevBootTime.Unix(), 1), "event:device-status/mac:112233445566/offline")
	fromEvent := suite.setEventDestination(fmt.Sprintf("%d-%d", prevBootTime.Unix(), 3), "event:device-status/mac:112233445566/offline")
	toEvent := suite.setEventDestination(fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2), "event-device-status/mac:112233445566/some-event")
	expectedEvents := suite.parseEvents(fromEvent, toEvent)
	parser := BootCycleParser(mockComparator, mockVal)
	results, err := parser.Parse(suite.Events, toEvent)
	suite.Equal(expectedEvents, results)
	suite.Nil(err)
}

func (suite *CycleTestSuite) TestInvalidEvents() {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	suite.Nil(err)

	futureBootTime := now.Add(1 * time.Hour)
	currentBootTime := now
	prevBootTime := now.Add(-1 * time.Hour)
	olderBootTime := now.Add(-2 * time.Hour)
	bootTimes := []testEventSetup{
		testEventSetup{
			bootTime:  currentBootTime,
			numEvents: 3,
		},
		testEventSetup{
			bootTime:  olderBootTime,
			numEvents: 3,
		},
		testEventSetup{
			bootTime:  prevBootTime,
			numEvents: 4,
		},
		testEventSetup{
			bootTime:  futureBootTime,
			numEvents: 2,
		},
	}

	suite.createEvents(bootTimes...)
	mockVal := new(mockValidator)
	mockComparator := new(mockComparator)
	fromEvent := suite.setEventDestination(fmt.Sprintf("%d-%d", prevBootTime.Unix(), 3), "event:device-status/mac:112233445566/reboot-pending")
	toEvent := suite.setEventDestination(fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2), "event-device-status/mac:112233445566/some-event")
	allErrors := make(map[string]error)
	for _, event := range suite.Events {
		err := fmt.Errorf("error %s", event.TransactionUUID)
		allErrors[event.TransactionUUID] = validation.EventWithError{
			Event:       event,
			OriginalErr: err,
		}

		mockVal.On("Valid", event).Return(false, err)
	}

	mockComparator.On("Compare", mock.Anything, mock.Anything).Return(false, nil)
	expectedEvents := suite.parseEvents(fromEvent, toEvent)
	parser := BootCycleParser(mockComparator, mockVal)
	results, err := parser.Parse(suite.Events, toEvent)
	suite.Equal(expectedEvents, results)
	var resultErrs validation.Errors
	suite.True(errors.As(err, &resultErrs))
	suite.Equal(len(results), len(resultErrs))
	for _, e := range resultErrs {
		var eventErr validation.EventWithError
		suite.True(errors.As(e, &eventErr))
		suite.Equal(eventErr, allErrors[eventErr.Event.TransactionUUID])
	}
}

func (suite *CycleTestSuite) TestInvalidComparator() {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	suite.Nil(err)

	futureBootTime := now.Add(1 * time.Hour)
	currentBootTime := now
	prevBootTime := now.Add(-1 * time.Hour)
	olderBootTime := now.Add(-2 * time.Hour)
	bootTimes := []testEventSetup{
		testEventSetup{
			bootTime:  currentBootTime,
			numEvents: 3,
		},
		testEventSetup{
			bootTime:  olderBootTime,
			numEvents: 3,
		},
		testEventSetup{
			bootTime:  prevBootTime,
			numEvents: 4,
		},
		testEventSetup{
			bootTime:  futureBootTime,
			numEvents: 2,
		},
	}

	suite.createEvents(bootTimes...)
	mockVal := new(mockValidator)
	mockComparator := new(mockComparator)
	testErr := errors.New("test")
	mockVal.On("Valid", mock.Anything).Return(true, nil)
	mockComparator.On("Compare", mock.Anything, mock.Anything).Return(true, testErr)
	toEvent := suite.setEventDestination(fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2), "event-device-status/mac:112233445566/some-event")
	parser := BootCycleParser(mockComparator, mockVal)
	results, err := parser.Parse(suite.Events, toEvent)
	suite.Empty(results)
	suite.True(errors.Is(err, testErr))
}

func (suite *CycleTestSuite) TestCurrentEventInvalidBootTime() {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	suite.Nil(err)

	futureBootTime := now.Add(1 * time.Hour)
	currentBootTime := now
	prevBootTime := now.Add(-1 * time.Hour)
	olderBootTime := now.Add(-2 * time.Hour)
	bootTimes := []testEventSetup{
		testEventSetup{
			bootTime:  currentBootTime,
			numEvents: 3,
		},
		testEventSetup{
			bootTime:  olderBootTime,
			numEvents: 3,
		},
		testEventSetup{
			bootTime:  prevBootTime,
			numEvents: 4,
		},
		testEventSetup{
			bootTime:  futureBootTime,
			numEvents: 2,
		},
	}

	suite.createEvents(bootTimes...)
	tests := []interpreter.Event{
		interpreter.Event{}, interpreter.Event{Metadata: map[string]string{interpreter.BootTimeKey: "-1"}},
	}

	mockVal := new(mockValidator)
	mockComparator := new(mockComparator)

	for _, event := range tests {
		parser := BootCycleParser(mockComparator, mockVal)
		results, err := parser.Parse(suite.Events, event)
		suite.Empty(results)
		var invalidBootTimeErr validation.InvalidBootTimeErr
		suite.True(errors.As(err, &invalidBootTimeErr))
	}

}

func TestBootCycleParser(t *testing.T) {
	suite.Run(t, new(CycleTestSuite))
}

func TestRebootEventsParser(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	rebootPendingEvent := interpreter.Event{
		TransactionUUID: "1",
		Destination:     "event:device-status/mac:112233445566/reboot-pending",
		Birthdate:       now.UnixNano(),
	}
	events := []interpreter.Event{
		interpreter.Event{
			TransactionUUID: "2",
			Destination:     "event:device-status/mac:112233445566/online",
			Birthdate:       now.Add(2 * time.Minute).UnixNano(),
		},
		interpreter.Event{
			TransactionUUID: "3",
			Destination:     "event:device-status/mac:112233445566/offline",
			Birthdate:       now.Add(3 * time.Minute).UnixNano(),
		},
		interpreter.Event{
			TransactionUUID: "-1",
			Destination:     "event:device-status/mac:112233445566/offline",
			Birthdate:       now.Add(-1 * time.Minute).UnixNano(),
		},
		interpreter.Event{
			TransactionUUID: "-2",
			Birthdate:       now.Add(-2 * time.Minute).UnixNano(),
		},
		interpreter.Event{
			TransactionUUID: "4",
			Destination:     "event:device-status/mac:112233445566/online",
			Birthdate:       now.Add(4 * time.Minute).UnixNano(),
		},
		interpreter.Event{
			TransactionUUID: "5",
			Birthdate:       now.Add(5 * time.Minute).UnixNano(),
		},
	}

	tests := []struct {
		description      string
		events           []interpreter.Event
		expectedEventIDs map[string]bool
	}{
		{
			description: "with reboot-pending event",
			events:      append(events, rebootPendingEvent),
			expectedEventIDs: map[string]bool{
				rebootPendingEvent.TransactionUUID: true,
				"2":                                true,
				"3":                                true,
				"4":                                true,
				"5":                                true,
			},
		},
		{
			description: "without reboot-pending event",
			events:      events,
			expectedEventIDs: map[string]bool{
				"3": true,
				"4": true,
				"5": true,
			},
		},
		{
			description: "empty list",
			events:      []interpreter.Event{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			cycle := rebootEventsParser(tc.events)
			assert.Equal(len(tc.expectedEventIDs), len(cycle))
			for _, event := range cycle {
				assert.True(tc.expectedEventIDs[event.TransactionUUID])
			}
		})
	}
}

type testEventValidation struct {
	event interpreter.Event
	valid bool
	err   error
}

func TestValidateEvents(t *testing.T) {
	assert := assert.New(t)
	mockVal := new(mockValidator)
	tests := []testEventValidation{
		testEventValidation{
			event: interpreter.Event{TransactionUUID: "1"},
			valid: false,
			err:   errors.New("test 1"),
		},
		testEventValidation{
			event: interpreter.Event{TransactionUUID: "2"},
			valid: true,
		},
		testEventValidation{
			event: interpreter.Event{TransactionUUID: "3"},
			valid: true,
		},
		testEventValidation{
			event: interpreter.Event{TransactionUUID: "4"},
			valid: false,
			err:   errors.New("test 2"),
		},
		testEventValidation{
			event: interpreter.Event{TransactionUUID: "5"},
			valid: false,
			err:   errors.New("test 5"),
		},
	}

	events := make([]interpreter.Event, len(tests))
	var allErrors validation.Errors
	for i, test := range tests {
		mockVal.On("Valid", test.event).Return(test.valid, test.err)
		events[i] = test.event
		if !test.valid {
			allErrors = append(allErrors, validation.EventWithError{
				Event:       test.event,
				OriginalErr: test.err,
			})
		}
	}

	err := validateEvents(events, mockVal)
	var allTestErrors validation.Errors
	assert.True(errors.As(err, &allTestErrors))
	assert.ElementsMatch(allErrors, allTestErrors)

	mockVal2 := new(mockValidator)
	mockVal2.On("Valid", mock.Anything).Return(true, nil)
	err = validateEvents(events, mockVal2)
	assert.Nil(err)
}
