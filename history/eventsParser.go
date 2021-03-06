/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package history

import (
	"sort"

	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

// sortFunc is a function type passed into the sort function.
type sortFunc func(a, b int) bool

func birthdateDescendingSortFunc(events []interpreter.Event) sortFunc {
	return func(a, b int) bool {
		return events[a].Birthdate > events[b].Birthdate
	}
}

func bootTimeDescendingSortFunc(events []interpreter.Event) sortFunc {
	return func(a, b int) bool {
		boottimeA, _ := events[a].BootTime()
		boottimeB, _ := events[b].BootTime()
		if boottimeA != boottimeB {
			return boottimeA > boottimeB
		}

		return events[a].Birthdate > events[b].Birthdate

	}
}

// EventsParserFunc is a function that returns the relevant events from a slice of events.
type EventsParserFunc func([]interpreter.Event, interpreter.Event) ([]interpreter.Event, error)

// Parse implements the EventsParser interface.
func (p EventsParserFunc) Parse(events []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
	return p(events, currentEvent)
}

// DefaultCycleParser runs each event in the history through the comparator and returns the entire
// history if no errors are found.
func DefaultCycleParser(comparator Comparator) EventsParserFunc {
	comparator = setComparator(comparator)
	return func(eventsHistory []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
		latestBootTime, err := currentEvent.BootTime()
		if err != nil || latestBootTime <= 0 {
			return []interpreter.Event{}, validation.InvalidBootTimeErr{OriginalErr: err}
		}

		var eventList []interpreter.Event
		for _, event := range eventsHistory {
			// If comparator returns true, it means we should stop parsing
			// because there is something wrong with currentEvent
			if bad, err := comparator.Compare(event, currentEvent); bad {
				return []interpreter.Event{}, err
			}

			// make sure event is not the current event
			if event.TransactionUUID != currentEvent.TransactionUUID {
				eventList = append(eventList, event)
			}
		}

		eventList = append(eventList, currentEvent)
		sort.Slice(eventList, birthdateDescendingSortFunc(eventList))

		return eventList, nil
	}
}

// RebootParser returns an EventsParser that takes in a list of events and returns a sorted subset of that list
// containing events that are relevant to the latest reboot. The slice starts with the last reboot-pending (if available) or last offline event
// and includes all events afterwards that have a birthdate less than the first fully-manageable event (if available) or the first
// operational event (if available) or the first online event of the current cycle. The returned slice is sorted from
// newest to oldest primarily by boot-time, and then by birthdate.
// RebootParser also runs the list of events through the comparator to see if the current event is valid.
func RebootParser(comparator Comparator) EventsParserFunc {
	comparator = setComparator(comparator)
	return func(eventsHistory []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
		lastCycle, currentCycle, err := parserHelper(eventsHistory, currentEvent, comparator)
		if err != nil {
			return []interpreter.Event{}, err
		}

		rebootStart := rebootStartParser(lastCycle)
		rebootEnd := rebootEndParser(currentCycle)
		cycle := append(rebootEnd, rebootStart...)
		return cycle, nil
	}
}

// RebootToCurrentParser returns an EventsParser that takes in a list of events and returns a sorted subset of that list
// containing events that are relevant to the latest reboot. The slice starts with the last reboot-pending (if available)
// or last offline event and includes all events afterwards that have a birthdate less than or equal to the current event.
// The returned slice is sorted from newest to oldest primarily by boot-time, and then by birthdate.
// RebootToCurrentParser also runs the list of events through the comparator to see if the current event is valid.
func RebootToCurrentParser(comparator Comparator) EventsParserFunc {
	comparator = setComparator(comparator)
	return func(eventsHistory []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
		lastCycle, currentCycle, err := parserHelper(eventsHistory, currentEvent, comparator)
		if err != nil {
			return []interpreter.Event{}, err
		}

		lastCycle = rebootStartParser(lastCycle)
		cycle := append(currentCycle, lastCycle...)
		return cycle, nil
	}
}

// LastCycleParser returns an EventsParser that takes in a list of events and returns a sorted subset
// of that list which includes all of the events with the boot-time of the previous cycle sorted from newest to oldest
// by birthdate. LastCycleParser also runs the list of events through the comparator to see if the current event is valid.
func LastCycleParser(comparator Comparator) EventsParserFunc {
	comparator = setComparator(comparator)
	return func(eventsHistory []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
		lastCycle, _, err := parserHelper(eventsHistory, currentEvent, comparator)
		if err != nil {
			return []interpreter.Event{}, err
		}

		return lastCycle, nil
	}
}

// LastCycleToCurrentParser returns an EventsParser that takes in a list of events and returns a sorted subset
// of that list. The slice includes all of the events with the boot-time of the previous cycle
// as well as all events with the latest boot-time that have a birthdate less than or equal to the current event.
// The returned slice is sorted from newest to oldest primarily by boot-time, and then by birthdate.
func LastCycleToCurrentParser(comparator Comparator) EventsParserFunc {
	comparator = setComparator(comparator)
	return func(eventsHistory []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
		lastCycle, currentCycle, err := parserHelper(eventsHistory, currentEvent, comparator)
		if err != nil {
			return []interpreter.Event{}, err
		}

		cycle := append(currentCycle, lastCycle...)
		return cycle, nil
	}
}

// CurrentCycleParser returns an EventsParser that takes in a list of events and returns a sorted subset
// of that list which includes all of the events with the boot-time of the current cycle sorted from newest to oldest
// by birthdate. CurrentCycleParser also runs the list of events through the comparator to see if the current event is valid.
func CurrentCycleParser(comparator Comparator) EventsParserFunc {
	comparator = setComparator(comparator)
	return func(eventsHistory []interpreter.Event, currentEvent interpreter.Event) ([]interpreter.Event, error) {
		currentCycle, err := getSameBootTimeEvents(eventsHistory, currentEvent, comparator)
		if err != nil {
			return []interpreter.Event{}, err
		}

		return currentCycle, nil
	}
}

// parserHelper takes in a list of events and returns two slices: one containing all of the events with the previous cycle's boot-time and
// another containing events with the latest boot-time and birthdate less than the currentEvent.
// It also runs all of the events in the events list through the comparator, and if the comparator returns true,
// parserHelper will stop and return two empty slices and the error returned by the comparator.
// The two slices are sorted from newest to oldest.
func parserHelper(events []interpreter.Event, currentEvent interpreter.Event, comparator Comparator) ([]interpreter.Event, []interpreter.Event, error) {
	latestBootTime, err := currentEvent.BootTime()
	if err != nil || latestBootTime <= 0 {
		return []interpreter.Event{}, []interpreter.Event{}, validation.InvalidBootTimeErr{OriginalErr: err}
	}

	var lastCycle []interpreter.Event
	var currentCycle []interpreter.Event
	var lastBoottime int64
	for _, event := range events {
		bootTime, _ := event.BootTime()
		if bootTime <= 0 {
			continue
		}

		// If comparator returns true, it means we should stop parsing
		// because there is something wrong with currentEvent
		if bad, err := comparator.Compare(event, currentEvent); bad {
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
		if bootTime == latestBootTime && event.Birthdate <= currentEvent.Birthdate && !sameEvent(event, currentEvent) {
			currentCycle = append(currentCycle, event)
		}
	}

	sort.Slice(lastCycle, birthdateDescendingSortFunc(lastCycle))

	currentCycle = append(currentCycle, currentEvent)
	sort.Slice(currentCycle, birthdateDescendingSortFunc(currentCycle))

	return lastCycle, currentCycle, nil
}

// getSameBootTimeEvents returns a list of events with the same boot-time as the currentEvent, along with the currentEvent.
// It also runs all of the events in the events list through the comparator, and if the comparator returns true,
// getSameBootTimeEvents will stop and return an empty slice and the error returned by the comparator.
// The slice is sorted from newest to oldest by birthdate.
func getSameBootTimeEvents(events []interpreter.Event, currentEvent interpreter.Event, comparator Comparator) ([]interpreter.Event, error) {
	latestBootTime, err := currentEvent.BootTime()
	if err != nil || latestBootTime <= 0 {
		return []interpreter.Event{}, validation.InvalidBootTimeErr{OriginalErr: err}
	}

	var currentCycle []interpreter.Event
	for _, event := range events {
		bootTime, _ := event.BootTime()
		if bootTime <= 0 {
			continue
		}

		// If comparator returns true, it means we should stop parsing
		// because there is something wrong with currentEvent
		if bad, err := comparator.Compare(event, currentEvent); bad {
			return []interpreter.Event{}, err
		}

		if bootTime == latestBootTime && !sameEvent(event, currentEvent) {
			currentCycle = append(currentCycle, event)
		}
	}

	currentCycle = append(currentCycle, currentEvent)
	sort.Slice(currentCycle, birthdateDescendingSortFunc(currentCycle))
	return currentCycle, nil
}

func sameEvent(eventA interpreter.Event, eventB interpreter.Event) bool {
	bootTimeA, _ := eventA.BootTime()
	bootTimeB, _ := eventB.BootTime()
	return bootTimeA == bootTimeB && eventA.Birthdate == eventB.Birthdate && eventA.TransactionUUID == eventB.TransactionUUID
}

// returns default comparator if comparator is nil
func setComparator(comparator Comparator) Comparator {
	if comparator == nil {
		comparator = DefaultComparator()
	}

	return comparator
}

// rebootStartParser is a helper function that takes in a list of events
// and returns a slice containing the last reboot-pending or offline event and any events that come after.
// If the slice does not contain a reboot-pending or offline event, an empty list is returned.
// Assumes that all events in the list have the same boot-time.
func rebootStartParser(events []interpreter.Event) []interpreter.Event {
	if len(events) == 0 {
		return events
	}

	sort.Slice(events, birthdateDescendingSortFunc(events))

	lastOfflineIndex := -1
	for i, event := range events {
		eventType, err := event.EventType()
		if err == nil {
			if eventType == interpreter.RebootPendingEventType {
				return events[:i+1]
			} else if eventType == interpreter.OfflineEventType && lastOfflineIndex == -1 {
				lastOfflineIndex = i
			}
		}
	}

	return events[:lastOfflineIndex+1]
}

// rebootEndParser is a helper function that takes in a list of events
// and returns a slice containing events before the first fully-manageable event. If a fully-manageable event
// doesn't exist, it looks for the first operational event, then online event. If all these events don't exist, it
// returns an empty list. Assumes that all events in the list have the same boot-time and that the order of the boot-cycle
// is: online, operational, fully-manageable.
func rebootEndParser(events []interpreter.Event) []interpreter.Event {
	if len(events) == 0 {
		return events
	}

	sort.Slice(events, birthdateDescendingSortFunc(events))
	operationalIndex := -1
	onlineIndex := -1
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		eventType, err := event.EventType()
		if err == nil {
			if eventType == interpreter.FullyManageableEventType {
				return events[i:]
			} else if eventType == interpreter.OperationalEventType && operationalIndex == -1 {
				operationalIndex = i
			} else if eventType == interpreter.OnlineEventType && onlineIndex == -1 {
				onlineIndex = i
			}
		}
	}

	if operationalIndex > -1 {
		return events[operationalIndex:]
	}

	if onlineIndex > -1 {
		return events[onlineIndex:]
	}

	return []interpreter.Event{}
}
