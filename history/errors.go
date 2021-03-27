package history

import (
	"fmt"

	"github.com/xmidt-org/interpreter"
)

// ComparatorErr is used when an error is found with a trigger event
// when comparing it to a another event in the history of events.
type ComparatorErr struct {
	OriginalErr     error
	ComparisonEvent interpreter.Event
}

func (e ComparatorErr) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("comparator error: %v", e.OriginalErr)
	}

	return "comparator error"
}

func (e ComparatorErr) Unwrap() error {
	return e.OriginalErr
}

// Event returns the event in history that caused the error to be thrown.
func (e ComparatorErr) Event() interpreter.Event {
	return e.ComparisonEvent
}

// EventFinderErr is an error used by EventFinder.
type EventFinderErr struct {
	OriginalErr error
}

func (e EventFinderErr) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("failed to find event: %v", e.OriginalErr)
	}

	return "failed to find event"
}

func (e EventFinderErr) Unwrap() error {
	return e.OriginalErr
}
