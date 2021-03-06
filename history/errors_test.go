package history

import (
	"testing"

	"github.com/xmidt-org/interpreter/validation"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/interpreter"
)

func TestEventCompareErr(t *testing.T) {
	const testTag validation.Tag = 1000
	testErr := testTaggedError{tag: testTag}
	tests := []struct {
		description   string
		err           ComparatorErr
		expectedErr   error
		expectedEvent interpreter.Event
		expectedTag   validation.Tag
	}{
		{
			description: "No underlying error or event",
			err:         ComparatorErr{},
		},
		{
			description: "Underlying error",
			err:         ComparatorErr{OriginalErr: testErr},
			expectedErr: testErr,
			expectedTag: testTag,
		},
		{
			description:   "Underlying event",
			err:           ComparatorErr{ComparisonEvent: interpreter.Event{Destination: "test-dest"}},
			expectedEvent: interpreter.Event{Destination: "test-dest"},
		},
		{
			description: "With Tag",
			err:         ComparatorErr{OriginalErr: testErr, ErrorTag: 2000},
			expectedErr: testErr,
			expectedTag: 2000,
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
			assert.Equal(tc.expectedTag, tc.err.Tag())
		})
	}
}

func TestEventFinderErr(t *testing.T) {
	const testTag validation.Tag = 1000
	testErr := testTaggedError{tag: testTag}
	tests := []struct {
		description string
		err         EventFinderErr
		expectedErr error
		expectedTag validation.Tag
	}{
		{
			description: "No underlying error or event",
			err:         EventFinderErr{},
		},
		{
			description: "Underlying error",
			err:         EventFinderErr{OriginalErr: testErr},
			expectedErr: testErr,
			expectedTag: testTag,
		},
		{
			description: "With Tag",
			err:         EventFinderErr{OriginalErr: testErr, ErrorTag: 2000},
			expectedErr: testErr,
			expectedTag: 2000,
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
			assert.Equal(tc.expectedTag, tc.err.Tag())
		})
	}
}

func TestCycleValidationErr(t *testing.T) {
	const testTag validation.Tag = 1000
	testErr := testTaggedError{tag: testTag}
	tests := []struct {
		description          string
		err                  CycleValidationErr
		expectedErr          error
		expectedTag          validation.Tag
		expectedFields       []string
		expectedDetailString string
	}{
		{
			description:          "No underlying error or fields",
			err:                  CycleValidationErr{},
			expectedDetailString: "unknown: []",
		},
		{
			description:          "Underlying error",
			err:                  CycleValidationErr{OriginalErr: testErr},
			expectedErr:          testErr,
			expectedTag:          testTag,
			expectedDetailString: "unknown: []",
		},
		{
			description:          "With Tag",
			err:                  CycleValidationErr{OriginalErr: testErr, ErrorTag: 2000},
			expectedErr:          testErr,
			expectedTag:          2000,
			expectedDetailString: "unknown: []",
		},
		{
			description:          "With Fields",
			err:                  CycleValidationErr{OriginalErr: testErr, ErrorTag: 2000, ErrorDetailValues: []string{"test", "test2"}, ErrorDetailKey: "descriptive key"},
			expectedErr:          testErr,
			expectedTag:          2000,
			expectedFields:       []string{"test", "test2"},
			expectedDetailString: "descriptive key: [test, test2]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			if tc.expectedErr != nil {
				assert.Contains(tc.err.Error(), tc.expectedErr.Error())
			}
			assert.Contains(tc.err.Error(), "cycle validation error")
			assert.Equal(tc.expectedErr, tc.err.Unwrap())
			assert.Equal(tc.expectedTag, tc.err.Tag())
			assert.ElementsMatch(tc.expectedFields, tc.err.Fields())
			assert.Contains(tc.err.ErrorDetails(), tc.expectedDetailString)
		})
	}

}
