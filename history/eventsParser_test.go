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

type eventToChange struct {
	eventID          string
	eventDestination string
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
	eventsCopy := make([]interpreter.Event, len(suite.Events))
	copy(eventsCopy, suite.Events)
	sort.Slice(eventsCopy, bootTimeDescendingSortFunc(eventsCopy))

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

func (suite *CycleTestSuite) parseEventsBy(addToList func(event interpreter.Event) bool) []interpreter.Event {
	var eventsCopy []interpreter.Event
	for _, event := range suite.Events {
		if addToList(event) {
			eventsCopy = append(eventsCopy, event)
		}
	}

	sort.Slice(eventsCopy, birthdateDescendingSortFunc(eventsCopy))
	return eventsCopy
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

func (suite *CycleTestSuite) clearEventDestinations() {
	for i := range suite.Events {
		suite.Events[i].Destination = ""
	}
}

func (suite *CycleTestSuite) TestParserHelperValid() {
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
	mockComparator := new(mockComparator)
	mockComparator.On("Compare", mock.Anything, mock.Anything).Return(false, nil)
	suite.setEventDestination(fmt.Sprintf("%d-%d", prevBootTime.Unix(), 3), "event:device-status/mac:112233445566/offline")
	toEvent := suite.setEventDestination(fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2), "event-device-status/mac:112233445566/some-event")
	expectedLastCycle := suite.parseEventsBy(func(e interpreter.Event) bool {
		boottime, _ := e.BootTime()
		return boottime == prevBootTime.Unix()
	})
	expectedCurrentCycle := suite.parseEventsBy(func(e interpreter.Event) bool {
		boottime, _ := e.BootTime()
		return (boottime == currentBootTime.Unix() && e.Birthdate <= toEvent.Birthdate) || e.TransactionUUID == toEvent.TransactionUUID
	})
	lastCycle, currentCycle, err := parserHelper(suite.Events, toEvent, mockComparator)
	suite.Equal(expectedLastCycle, lastCycle)
	suite.Equal(expectedCurrentCycle, currentCycle)
	suite.Nil(err)
}

func (suite *CycleTestSuite) TestParserHelperErr() {
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
	mockComparator := new(mockComparator)
	testErr := errors.New("test")
	mockComparator.On("Compare", mock.Anything, mock.Anything).Return(true, testErr)
	toEvent := suite.setEventDestination(fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2), "event-device-status/mac:112233445566/some-event")
	lastCycle, currentCycle, err := parserHelper(suite.Events, toEvent, mockComparator)
	suite.Empty(lastCycle)
	suite.Empty(currentCycle)
	suite.True(errors.Is(err, testErr))
}

func (suite *CycleTestSuite) TestParsersValid() {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	suite.Nil(err)

	mockComparator := new(mockComparator)
	mockComparator.On("Compare", mock.Anything, mock.Anything).Return(false, nil)

	futureBootTime := now.Add(1 * time.Hour)
	currentBootTime := now
	prevBootTime := now.Add(-1 * time.Hour)
	olderBootTime := now.Add(-2 * time.Hour)

	bootTimes := []testEventSetup{
		testEventSetup{
			bootTime:  currentBootTime,
			numEvents: 4,
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

	tests := []struct {
		description   string
		parser        EventsParserFunc
		startingEvent eventToChange
		endingEvent   eventToChange
		currentEvent  eventToChange
		parseFunc     func(interpreter.Event) bool
	}{
		{
			description:   "last cycle parser",
			parser:        LastCycleParser(mockComparator),
			startingEvent: eventToChange{eventID: fmt.Sprintf("%d-%d", prevBootTime.Unix(), 1)},
			endingEvent:   eventToChange{eventID: fmt.Sprintf("%d-%d", prevBootTime.Unix(), 4)},
			currentEvent:  eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2)},
			parseFunc: func(e interpreter.Event) bool {
				boottime, _ := e.BootTime()
				return boottime == prevBootTime.Unix()
			},
		},
		{
			description:   "last cycle to current parser",
			parser:        LastCycleToCurrentParser(mockComparator),
			startingEvent: eventToChange{eventID: fmt.Sprintf("%d-%d", prevBootTime.Unix(), 1)},
			endingEvent:   eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2)},
			currentEvent:  eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2)},
		},
		{
			description:   "current cycle parser",
			parser:        CurrentCycleParser(mockComparator),
			startingEvent: eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 1)},
			endingEvent:   eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2)},
			currentEvent:  eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2)},
		},
		{
			description: "reboot parser-fully-manageable",
			parser:      RebootParser(mockComparator),
			startingEvent: eventToChange{
				eventID:          fmt.Sprintf("%d-%d", prevBootTime.Unix(), 2),
				eventDestination: "event:device-status/mac:112233445566/reboot-pending",
			},
			endingEvent: eventToChange{
				eventID:          fmt.Sprintf("%d-%d", currentBootTime.Unix(), 3),
				eventDestination: "event:device-status/mac:112233445566/fully-manageable",
			},
			currentEvent: eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 4)},
		},
		{
			description: "reboot parser-operational event",
			parser:      RebootParser(mockComparator),
			startingEvent: eventToChange{
				eventID:          fmt.Sprintf("%d-%d", prevBootTime.Unix(), 2),
				eventDestination: "event:device-status/mac:112233445566/reboot-pending",
			},
			endingEvent: eventToChange{
				eventID:          fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2),
				eventDestination: "event:device-status/mac:112233445566/operational",
			},
			currentEvent: eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 4)},
		},
		{
			description: "reboot parser-online event",
			parser:      RebootParser(mockComparator),
			startingEvent: eventToChange{
				eventID:          fmt.Sprintf("%d-%d", prevBootTime.Unix(), 2),
				eventDestination: "event:device-status/mac:112233445566/reboot-pending",
			},
			endingEvent: eventToChange{
				eventID:          fmt.Sprintf("%d-%d", currentBootTime.Unix(), 1),
				eventDestination: "event:device-status/mac:112233445566/online",
			},
			currentEvent: eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 3)},
		},
		{
			description: "reboot parser-no reboot pending event",
			parser:      RebootParser(mockComparator),
			startingEvent: eventToChange{
				eventID:          fmt.Sprintf("%d-%d", prevBootTime.Unix(), 2),
				eventDestination: "event:device-status/mac:112233445566/offline",
			},
			endingEvent: eventToChange{
				eventID:          fmt.Sprintf("%d-%d", currentBootTime.Unix(), 3),
				eventDestination: "event:device-status/mac:112233445566/fully-manageable",
			},
			currentEvent: eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 4)},
		},
		{
			description: "reboot to current parser-no reboot pending event",
			parser:      RebootToCurrentParser(mockComparator),
			startingEvent: eventToChange{
				eventID:          fmt.Sprintf("%d-%d", prevBootTime.Unix(), 2),
				eventDestination: "event:device-status/mac:112233445566/offline",
			},
			endingEvent: eventToChange{
				eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 3),
			},
			currentEvent: eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 3)},
		},
		{
			description: "reboot to current parser with reboot pending event",
			parser:      RebootToCurrentParser(mockComparator),
			startingEvent: eventToChange{
				eventID:          fmt.Sprintf("%d-%d", prevBootTime.Unix(), 2),
				eventDestination: "event:device-status/mac:112233445566/reboot-pending",
			},
			endingEvent: eventToChange{
				eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 3),
			},
			currentEvent: eventToChange{eventID: fmt.Sprintf("%d-%d", currentBootTime.Unix(), 3)},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.description, func() {
			suite.clearEventDestinations()
			currentEvent := suite.setEventDestination(tc.currentEvent.eventID, tc.currentEvent.eventDestination)
			startingEvent := suite.setEventDestination(tc.startingEvent.eventID, tc.startingEvent.eventDestination)
			endingEvent := suite.setEventDestination(tc.endingEvent.eventID, tc.endingEvent.eventDestination)
			var expectedCycle []interpreter.Event
			if tc.parseFunc != nil {
				expectedCycle = suite.parseEventsBy(tc.parseFunc)
			} else {
				expectedCycle = suite.parseEvents(endingEvent, startingEvent)
			}
			cycle, err := tc.parser.Parse(suite.Events, currentEvent)
			suite.Equal(expectedCycle, cycle)
			suite.Nil(err)
		})

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
	mockComparator := new(mockComparator)
	testErr := errors.New("test")
	mockComparator.On("Compare", mock.Anything, mock.Anything).Return(true, testErr)
	toEvent := suite.setEventDestination(fmt.Sprintf("%d-%d", currentBootTime.Unix(), 2), "event-device-status/mac:112233445566/some-event")
	parsers := []EventsParserFunc{
		RebootParser(mockComparator),
		RebootToCurrentParser(mockComparator),
		LastCycleParser(mockComparator),
		CurrentCycleParser(mockComparator),
		LastCycleToCurrentParser(mockComparator),
	}

	for _, parser := range parsers {
		results, err := parser.Parse(suite.Events, toEvent)
		suite.Empty(results)
		suite.True(errors.Is(err, testErr))
	}

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

	mockComparator := new(mockComparator)
	parsers := []EventsParserFunc{
		CurrentCycleParser(mockComparator),
		RebootParser(mockComparator),
		RebootToCurrentParser(mockComparator),
		LastCycleParser(mockComparator),
		LastCycleToCurrentParser(mockComparator),
	}

	for _, event := range tests {
		for _, parser := range parsers {
			results, err := parser.Parse(suite.Events, event)
			suite.Empty(results)
			var invalidBootTimeErr validation.InvalidBootTimeErr
			suite.True(errors.As(err, &invalidBootTimeErr))
		}

	}

}

func TestParsers(t *testing.T) {
	suite.Run(t, new(CycleTestSuite))
}

func TestRebootStartParser(t *testing.T) {
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
			description: "without reboot-pending or offline event",
			events: []interpreter.Event{
				interpreter.Event{
					TransactionUUID: "2",
					Destination:     "event:device-status/mac:112233445566/online",
					Birthdate:       now.Add(2 * time.Minute).UnixNano(),
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
			cycle := rebootStartParser(tc.events)
			assert.Equal(len(tc.expectedEventIDs), len(cycle))
			for _, event := range cycle {
				assert.True(tc.expectedEventIDs[event.TransactionUUID])
			}
		})
	}
}

func TestRebootEndParser(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	fullyManageableEvent := interpreter.Event{
		TransactionUUID: "fully-manageable",
		Destination:     "event:device-status/mac:112233445566/fully-manageable",
		Birthdate:       now.UnixNano(),
	}
	operationalEvent := interpreter.Event{
		TransactionUUID: "operational",
		Destination:     "event:device-status/mac:112233445566/operational",
		Birthdate:       now.Add(-1 * time.Minute).UnixNano(),
	}
	onlineEvent := interpreter.Event{
		TransactionUUID: "online",
		Destination:     "event:device-status/mac:112233445566/online",
		Birthdate:       now.Add(-2 * time.Minute).UnixNano(),
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
			TransactionUUID: "-3",
			Destination:     "event:device-status/mac:112233445566/offline",
			Birthdate:       now.Add(-3 * time.Minute).UnixNano(),
		},
		interpreter.Event{
			TransactionUUID: "-4",
			Birthdate:       now.Add(-4 * time.Minute).UnixNano(),
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
			description: "with fully-manageable event",
			events:      append(events, fullyManageableEvent, onlineEvent, operationalEvent),
			expectedEventIDs: map[string]bool{
				fullyManageableEvent.TransactionUUID: true,
				onlineEvent.TransactionUUID:          true,
				operationalEvent.TransactionUUID:     true,
				"-3":                                 true,
				"-4":                                 true,
			},
		},
		{
			description: "with operational event",
			events:      append(events, onlineEvent, operationalEvent),
			expectedEventIDs: map[string]bool{
				onlineEvent.TransactionUUID:      true,
				operationalEvent.TransactionUUID: true,
				"-3":                             true,
				"-4":                             true,
			},
		},
		{
			description: "with online event",
			events:      append(events, onlineEvent),
			expectedEventIDs: map[string]bool{
				onlineEvent.TransactionUUID: true,
				"-3":                        true,
				"-4":                        true,
			},
		},
		{
			description: "no online, operational, fully-manageable event",
			events: []interpreter.Event{
				interpreter.Event{
					TransactionUUID: "2",
					Destination:     "event:device-status/mac:112233445566/offline",
					Birthdate:       now.Add(2 * time.Minute).UnixNano(),
				},
				interpreter.Event{
					TransactionUUID: "4",
					Destination:     "event:device-status/mac:112233445566/offline",
					Birthdate:       now.Add(4 * time.Minute).UnixNano(),
				},
				interpreter.Event{
					TransactionUUID: "5",
					Birthdate:       now.Add(5 * time.Minute).UnixNano(),
				},
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
			cycle := rebootEndParser(tc.events)
			assert.Equal(len(tc.expectedEventIDs), len(cycle))
			for _, event := range cycle {
				assert.True(tc.expectedEventIDs[event.TransactionUUID])
			}
		})
	}
}

func TestSetComparator(t *testing.T) {
	assert := assert.New(t)
	comparator := setComparator(nil)
	assert.NotNil(comparator)
	match, err := comparator.Compare(interpreter.Event{}, interpreter.Event{})
	assert.False(match)
	assert.Nil(err)
}
