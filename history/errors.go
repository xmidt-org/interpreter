package history

import (
	"fmt"

	"github.com/xmidt-org/interpreter"
)

// EventFinderErr is used when an error is found with a trigger event
// when comparing it to a list of events in history
type EventFinderErr struct {
	OriginalErr     error
	ComparisonEvent interpreter.Event
}

func (e EventFinderErr) Error() string {
	if e.OriginalErr != nil {
		return fmt.Sprintf("history comparison: invalid event. reason: %v", e.OriginalErr)
	}

	return "history comparison: invalid event."
}

func (e EventFinderErr) Unwrap() error {
	return e.OriginalErr
}

// ComparisonEvent returns the event in history that caused the error to be thrown.
func (e EventFinderErr) Event() interpreter.Event {
	return e.ComparisonEvent
}
