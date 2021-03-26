package history

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/interpreter"
)

func TestEventCompareErr(t *testing.T) {
	testErr := errors.New("test error")
	tests := []struct {
		description   string
		err           EventCompareErr
		expectedErr   error
		expectedEvent interpreter.Event
	}{
		{
			description: "No underlying error or event",
			err:         EventCompareErr{},
		},
		{
			description: "Underlying error",
			err:         EventCompareErr{OriginalErr: testErr},
			expectedErr: testErr,
		},
		{
			description:   "Underlying event",
			err:           EventCompareErr{ComparisonEvent: interpreter.Event{Destination: "test-dest"}},
			expectedEvent: interpreter.Event{Destination: "test-dest"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			if tc.expectedErr != nil {
				assert.Contains(tc.err.Error(), tc.expectedErr.Error())
			}
			assert.Contains(tc.err.Error(), "history comparison: invalid event")
			assert.Equal(tc.expectedErr, tc.err.Unwrap())
			assert.Equal(tc.expectedEvent, tc.err.Event())
		})
	}
}

func TestEventFinderErr(t *testing.T) {
	testErr := errors.New("test error")
	tests := []struct {
		description string
		err         EventFinderErr
		expectedErr error
	}{
		{
			description: "No underlying error or event",
			err:         EventFinderErr{},
		},
		{
			description: "Underlying error",
			err:         EventFinderErr{OriginalErr: testErr},
			expectedErr: testErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			if tc.expectedErr != nil {
				assert.Contains(tc.err.Error(), tc.expectedErr.Error())
			}
			assert.Contains(tc.err.Error(), "failed to find event")
			assert.Equal(tc.expectedErr, tc.err.Unwrap())
		})
	}
}
