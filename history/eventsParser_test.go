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

// for debugging purposes
func printEventArray(events []interpreter.Event, description string) {
	fmt.Printf("%s----------------------------------------\n", description)
	for _, event := range events {
		fmt.Printf("Destination: %s, TransactionID: %s\n", event.Destination, event.TransactionUUID)
	}
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
	results, err := parser(suite.Events, toEvent)
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
	results, err := parser(suite.Events, toEvent)
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
	results, err := parser(suite.Events, toEvent)
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
	results, err := parser(suite.Events, toEvent)
	suite.Empty(results)
	suite.True(errors.Is(err, testErr))
}

func TestBootCycleParser(t *testing.T) {
	suite.Run(t, new(CycleTestSuite))
}

type test struct {
	currentEvent   interpreter.Event
	comparator     Comparator
	validator      validation.Validator
	eventsList     []interpreter.Event
	expectedEvents []string
	expectedErr    error
}

func testBootCycleParserHelper(t *testing.T, testParams test) {
	assert := assert.New(t)
	parser := BootCycleParser(testParams.comparator, testParams.validator)
	events, err := parser(testParams.eventsList, testParams.currentEvent)
	assert.ElementsMatch(events, testParams.expectedEvents)
	assert.Equal(testParams.expectedErr, err)
}

func TestRebootEventsParser(t *testing.T) {
	assert := assert.New(t)
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(err)
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

	expectedEventIDs := map[string]bool{
		rebootPendingEvent.TransactionUUID: true,
		"2":                                true,
		"3":                                true,
		"4":                                true,
		"5":                                true,
	}

	testEvents := append(events, rebootPendingEvent)
	cycle := rebootEventsParser(testEvents)
	assert.Equal(len(expectedEventIDs), len(cycle))
	for _, event := range cycle {
		assert.True(expectedEventIDs[event.TransactionUUID])
	}

	expectedEventIDs = map[string]bool{
		"3": true,
		"4": true,
		"5": true,
	}
	cycle2 := rebootEventsParser(events)
	assert.Equal(len(expectedEventIDs), len(cycle2))
	for _, event := range cycle2 {
		assert.True(expectedEventIDs[event.TransactionUUID])
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

	var events []interpreter.Event
	var allErrors validation.Errors
	for _, test := range tests {
		mockVal.On("Valid", test.event).Return(test.valid, test.err)
		events = append(events, test.event)
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
