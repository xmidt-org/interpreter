package history

import (
	"errors"
	"regexp"

	"github.com/xmidt-org/interpreter"
)

var (
	errNewerBootTime  = errors.New("newer boot-time found")
	errDuplicateEvent = errors.New("duplicate event found")
)

// Comparator compares two events and returns a boolean indicating if the condition has been matched.
// The comparator will also return an error when it deems appropriate.
type Comparator interface {
	Compare(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error)
}

// ComparatorFunc is a function that compares two events.
type ComparatorFunc func(interpreter.Event, interpreter.Event) (bool, error)

// Compare runs the ComparatorFunc, making a ComparatorFunc a Comparator
func (c ComparatorFunc) Compare(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error) {
	return c(baseEvent, newEvent)
}

// Comparators are a list of objects that implement the Comparator interface
type Comparators []Comparator

// Compare runs through a list of Comparators and compares two events using
// each comparator. Returns true and an error at the first
// comparator that matches the condition that the comparator is looking for.
func (c Comparators) Compare(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error) {
	for _, comparator := range c {
		if match, err := comparator.Compare(baseEvent, newEvent); match {
			return true, err
		}
	}
	return false, nil
}

// OlderBootTimeComparator returns a ComparatorFunc to check and see if newEvent's boot-time is
// less than the baseEvent's boot-time. If it is, it returns true and an error.
// OlderBootTimeComparator assumes that newEvent has a valid boot-time
// and does not do any error-checking of newEvent's boot-time.
func OlderBootTimeComparator() ComparatorFunc {
	return func(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error) {
		// baseEvent is newEvent, no need to compare boot-times
		if baseEvent.TransactionUUID == newEvent.TransactionUUID {
			return false, nil
		}

		latestBootTime, _ := newEvent.BootTime()
		bootTime, err := baseEvent.BootTime()
		if err != nil || bootTime <= 0 {
			return false, nil
		}

		// if this event has a boot-time more recent than the latest one, return an error
		if bootTime > latestBootTime {
			return true, EventCompareErr{OriginalErr: errNewerBootTime, ComparisonEvent: baseEvent}
		}

		return false, nil

	}
}

// DuplicateEventComparator returns a ComparatorFunc to check and see if newEvent is a duplicate. A duplicate event
// in this case is defined as sharing the same event type and boot-time as the base event while having a birthdate
// that is equal to or newer than baseEvent's birthdate. If newEvent is found to be a duplicate, it returns true and
// an error. DuplicateEventComparator checks that newEvent and baseEvent match the eventType.
// It assumes that newEvent has a valid boot-time and does not do any error-checking of newEvent's boot-time.
func DuplicateEventComparator(eventType *regexp.Regexp) ComparatorFunc {
	return func(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error) {
		// baseEvent is newEvent, no need to compare boot-times
		if baseEvent.TransactionUUID == newEvent.TransactionUUID {
			return false, nil
		}

		// see if event is the type we are looking for
		if eventType.MatchString(baseEvent.Destination) && eventType.MatchString(newEvent.Destination) {
			latestBootTime, _ := newEvent.BootTime()
			bootTime, err := baseEvent.BootTime()
			if err != nil || bootTime <= 0 {
				return false, nil
			}

			// If the boot-time is the same as the latestBootTime, and the birthdate is older or equal,
			// this means that newEvent is a duplicate.
			if bootTime == latestBootTime && baseEvent.Birthdate <= newEvent.Birthdate {
				return true, EventCompareErr{OriginalErr: errDuplicateEvent, ComparisonEvent: baseEvent}
			}
		}

		return false, nil
	}
}
