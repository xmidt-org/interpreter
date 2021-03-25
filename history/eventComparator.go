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

// Comparator compares two events and returns a boolean indicating if the second event is valid or not.
// If invalid, an error is also returned.
type Comparator interface {
	Compare(interpreter.Event, interpreter.Event) (bool, error)
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
// each comparator. Returns false and an error at the first
// comparator that deems the newEvent invalid.
func (c Comparators) Compare(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error) {
	for _, comparator := range c {
		if valid, err := comparator.Compare(baseEvent, newEvent); !valid {
			return false, err
		}
	}
	return true, nil
}

// NewestBootTimeComparator returns a ComparatorFunc to check and see if baseEvent's boot-time is
// less than or equal to the newEvent's boot-time. NewestBootTimeComparator assumes that newEvent
// has a valid boot-time and does not do any error-checking of newEvent's boot-time.
func NewestBootTimeComparator() ComparatorFunc {
	return func(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error) {
		// baseEvent is newEvent, no need to compare boot-times
		if baseEvent.TransactionUUID == newEvent.TransactionUUID {
			return true, nil
		}

		latestBootTime, _ := newEvent.BootTime()
		bootTime, err := baseEvent.BootTime()
		if err != nil || bootTime <= 0 {
			return true, nil
		}

		// if this event has a boot-time more recent than the latest one, return an error
		if bootTime > latestBootTime {
			return false, EventCompareErr{OriginalErr: errNewerBootTime, ComparisonEvent: baseEvent}
		}

		return true, nil

	}
}

// UniqueEventComparator returns a ComparatorFunc to check and see if newEvent is unique, meaning that
// it does not share the same destination type and boot-time as baseEvent.
// UniqueEventComparator assumes that newEvent has a valid boot-time
// and does not do any error-checking of newEvent's boot-time.
func UniqueEventComparator(eventType *regexp.Regexp) ComparatorFunc {
	return func(baseEvent interpreter.Event, newEvent interpreter.Event) (bool, error) {
		// baseEvent is newEvent, no need to compare boot-times
		if baseEvent.TransactionUUID == newEvent.TransactionUUID {
			return true, nil
		}

		// see if event is the type we are looking for
		if eventType.MatchString(baseEvent.Destination) {
			latestBootTime, _ := newEvent.BootTime()
			bootTime, err := baseEvent.BootTime()
			if err != nil || bootTime <= 0 {
				return true, nil
			}

			// If the boot-time is the same as the latestBootTime, and the birthdate is older or equal,
			// this means that newEvent is a duplicate.
			if bootTime == latestBootTime && baseEvent.Birthdate <= newEvent.Birthdate {
				return false, EventCompareErr{OriginalErr: errDuplicateEvent, ComparisonEvent: baseEvent}
			}
		}

		return true, nil
	}
}
