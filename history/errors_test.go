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
		err           ComparatorErr
		expectedErr   error
		expectedEvent interpreter.Event
		expectedLabel string
	}{
		{
			description:   "No underlying error or event",
			err:           ComparatorErr{},
			expectedLabel: comparatorErrLabel,
		},
		{
			description:   "Underlying error",
			err:           ComparatorErr{OriginalErr: testErr},
			expectedErr:   testErr,
			expectedLabel: comparatorErrLabel,
		},
		{
			description:   "Underlying event",
			err:           ComparatorErr{ComparisonEvent: interpreter.Event{Destination: "test-dest"}},
			expectedEvent: interpreter.Event{Destination: "test-dest"},
			expectedLabel: comparatorErrLabel,
		},
		{
			description:   "With Label",
			err:           ComparatorErr{OriginalErr: testErr, ErrLabel: "test_error"},
			expectedErr:   testErr,
			expectedLabel: "test_error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			if tc.expectedErr != nil {
				assert.Contains(tc.err.Error(), tc.expectedErr.Error())
			}
			assert.Contains(tc.err.Error(), "comparator error")
			assert.Equal(tc.expectedErr, tc.err.Unwrap())
			assert.Equal(tc.expectedEvent, tc.err.Event())
			assert.Equal(tc.expectedLabel, tc.err.ErrorLabel())
		})
	}
}

func TestEventFinderErr(t *testing.T) {
	testErr := errors.New("test error")
	tests := []struct {
		description   string
		err           EventFinderErr
		expectedErr   error
		expectedLabel string
	}{
		{
			description:   "No underlying error or event",
			err:           EventFinderErr{},
			expectedLabel: finderErrLabel,
		},
		{
			description:   "Underlying error",
			err:           EventFinderErr{OriginalErr: testErr},
			expectedErr:   testErr,
			expectedLabel: finderErrLabel,
		},
		{
			description:   "With Label",
			err:           EventFinderErr{OriginalErr: testErr, ErrLabel: "test_error"},
			expectedErr:   testErr,
			expectedLabel: "test_error",
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
			assert.Equal(tc.expectedLabel, tc.err.ErrorLabel())
		})
	}
}
