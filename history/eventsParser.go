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

// BootCycleParser returns an EventsParser that takes in a list of events and returns a slice of events
// starting with the last reboot-pending (if available) or last offline event up to events from the current
// cycle that have a birthdate before the current event. The returned slice is sorted from oldest to newest
// primarily by boot-time, and then by birthdate. BootCycleParser also runs the list of events through the eventValidator
// and returns an error containing all of the invalid events with their corresponding errors.
func BootCycleParser(comparator Comparator, eventValidator validation.Validator) EventsParserFunc {
	return func(eventsHistory []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
		latestBootTime, err := currentEvent.BootTime()
		if err != nil || latestBootTime <= 0 {
			return []interpreter.Event{}, validation.InvalidBootTimeErr{OriginalErr: err}
		}

		var lastCycle []interpreter.Event
		var currentCycle []interpreter.Event
		var lastBoottime int64
		for _, event := range eventsHistory {
			bootTime, err := event.BootTime()
			if err != nil || bootTime == 0 {
				continue
			}

			// If comparator returns true, it means we should stop parsing
			// because there is something wrong with currentEvent
			if match, err := comparator.Compare(event, currentEvent); match {
				return []interpreter.Event{}, err
			}

			if bootTime > lastBoottime && bootTime < latestBootTime {
				lastBoottime = bootTime
				lastCycle = nil
			}

			if bootTime == lastBoottime {
				lastCycle = append(lastCycle, event)
			}

			if bootTime == latestBootTime && event.Birthdate < currentEvent.Birthdate {
				currentCycle = append(currentCycle, event)
			}
		}

		lastCycle = rebootEventsParser(lastCycle)
		currentCycle = append(currentCycle, currentEvent)
		sort.Slice(currentCycle, func(a, b int) bool {
			return currentCycle[a].Birthdate < currentCycle[b].Birthdate
		})

		cycle := append(lastCycle, currentCycle...)
		errs := validateEvents(cycle, eventValidator)
		return cycle, errs
	}
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
