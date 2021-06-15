package history

import (
	"sort"

	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

// EventsParserFunc is a function that returns the relevant events from a slice of events.
type EventsParserFunc func([]interpreter.Event, interpreter.Event) ([]interpreter.Event, error)

// Parse implements the EventsParser interface.
func (p EventsParserFunc) Parse(events []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
	return p(events, currentEvent)
}

// RebootParser returns an EventsParser that takes in a list of events and returns a slice of events that are
// relevant to the latest reboot. The slice starts with the last reboot-pending (if available) or last offline event
// and includes all events afterwards that have a birthdate less than or equal to the current event.
// The returned slice is sorted from oldest to newest primarily by boot-time, and then by birthdate.
// RebootParser also runs the list of events through the eventValidator
// and returns an error containing all of the invalid events with their corresponding errors.
func RebootParser(comparator Comparator, eventValidator validation.Validator) EventsParserFunc {
	comparator, eventValidator = setComparatorValidator(comparator, eventValidator)
	return func(eventsHistory []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
		lastCycle, currentCycle, err := parserHelper(eventsHistory, currentEvent, comparator)
		if err != nil {
			return []interpreter.Event{}, err
		}

		lastCycle = rebootEventsParser(lastCycle)
		cycle := append(lastCycle, currentCycle...)
		errs := validateEvents(cycle, eventValidator)
		return cycle, errs
	}
}

// LastCycleParser returns an EventsParser that takes in a list of events and returns a slice of events that includes
// all of the events with the boot-time of the previous cycle sorted from oldest to newest by birthdate.
// LastCycleParser also runs the list of events through the eventValidator
// and returns an error containing all of the invalid events with their corresponding errors.
func LastCycleParser(comparator Comparator, eventValidator validation.Validator) EventsParserFunc {
	comparator, eventValidator = setComparatorValidator(comparator, eventValidator)
	return func(eventsHistory []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
		lastCycle, _, err := parserHelper(eventsHistory, currentEvent, comparator)
		if err != nil {
			return []interpreter.Event{}, err
		}

		errs := validateEvents(lastCycle, eventValidator)
		return lastCycle, errs
	}
}

// LastCycleToCurrentParser returns an EventsParser that takes in a list of events and returns a slice of events.
// The slice includes all of the events with the boot-time of the previous cycle as well as all events with the latest boot-time
// that have a birthdate less than or equal to the current event.
// The returned slice is sorted from oldest to newest primarily by boot-time, and then by birthdate.
// LastCycleToCurrentParser also runs the list of events through the eventValidator
// and returns an error containing all of the invalid events with their corresponding errors.
func LastCycleToCurrentParser(comparator Comparator, eventValidator validation.Validator) EventsParserFunc {
	comparator, eventValidator = setComparatorValidator(comparator, eventValidator)
	return func(eventsHistory []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
		lastCycle, currentCycle, err := parserHelper(eventsHistory, currentEvent, comparator)
		if err != nil {
			return []interpreter.Event{}, err
		}

		cycle := append(lastCycle, currentCycle...)
		errs := validateEvents(cycle, eventValidator)
		return cycle, errs
	}
}

// parserHelper takes in a list of events and returns two slices: one containing all of the events with the previous cycle's boot-time and
// another containing events with the latest boot-time. It also runs all of the events in the events list through the comparator, and if the
// comparator returns true, parserHelper will stop and return two empty slices and the error returned by the comparator. The two slices are sorted
// from oldest to newest.
func parserHelper(events []interpreter.Event, currentEvent interpreter.Event, comparator Comparator) ([]interpreter.Event, []interpreter.Event, error) {
	latestBootTime, err := currentEvent.BootTime()
	if err != nil || latestBootTime <= 0 {
		return []interpreter.Event{}, []interpreter.Event{}, validation.InvalidBootTimeErr{OriginalErr: err}
	}

	var lastCycle []interpreter.Event
	var currentCycle []interpreter.Event
	var lastBoottime int64
	for _, event := range events {
		bootTime, err := event.BootTime()
		if err != nil || bootTime == 0 {
			continue
		}

		// If comparator returns true, it means we should stop parsing
		// because there is something wrong with currentEvent
		if match, err := comparator.Compare(event, currentEvent); match {
			return []interpreter.Event{}, []interpreter.Event{}, err
		}

		if bootTime > lastBoottime && bootTime < latestBootTime {
			lastBoottime = bootTime
			lastCycle = nil
		}

		if bootTime == lastBoottime {
			lastCycle = append(lastCycle, event)
		}

		// make sure event is not the current event
		if bootTime == latestBootTime && event.Birthdate < currentEvent.Birthdate && event.TransactionUUID != currentEvent.TransactionUUID {
			currentCycle = append(currentCycle, event)
		}
	}

	sort.Slice(lastCycle, func(a, b int) bool {
		return lastCycle[a].Birthdate < lastCycle[b].Birthdate
	})

	currentCycle = append(currentCycle, currentEvent)
	sort.Slice(currentCycle, func(a, b int) bool {
		return currentCycle[a].Birthdate < currentCycle[b].Birthdate
	})

	return lastCycle, currentCycle, nil
}

// returns default comparator and validator
func setComparatorValidator(comparator Comparator, eventValidator validation.Validator) (Comparator, validation.Validator) {
	if comparator == nil {
		comparator = ComparatorFunc(func(interpreter.Event, interpreter.Event) (bool, error) {
			return false, nil
		})
	}

	if eventValidator == nil {
		eventValidator = validation.ValidatorFunc(func(interpreter.Event) (bool, error) {
			return true, nil
		})
	}

	return comparator, eventValidator
}

func validateEvents(events []interpreter.Event, eventValidator validation.Validator) error {
	var allErrors validation.Errors
	for _, event := range events {
		if valid, err := eventValidator.Valid(event); !valid {
			allErrors = append(allErrors, validation.EventWithError{
				Event:       event,
				OriginalErr: err,
			})
		}
	}

	if len(allErrors) == 0 {
		return nil
	}

	return allErrors
}

// rebootEventsParser is a helper function that takes in a list of events
// and returns a slice containing the last reboot-pending or offline event and any events that come after.
// Assumes that all events in the list have the same boot-time.
func rebootEventsParser(events []interpreter.Event) []interpreter.Event {
	if len(events) == 0 {
		return events
	}

	sort.Slice(events, func(a, b int) bool {
		return events[a].Birthdate < events[b].Birthdate
	})

	var lastOfflineIndex int
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		eventType, err := event.EventType()
		if err == nil {
			if eventType == interpreter.RebootPendingEventType {
				return events[i:]
			} else if eventType == interpreter.OfflineEventType && i > lastOfflineIndex {
				lastOfflineIndex = i
			}
		}
	}

	return events[lastOfflineIndex:]
}
