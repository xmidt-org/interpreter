package validation

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/xmidt-org/interpreter"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		description        string
		errList            []error
		expectedTags       []Tag
		expectedTag        Tag
		expectedUniqueTags []Tag
	}{
		{
			description:  "no tags",
			errList:      []error{errors.New("test"), errors.New("test2"), errors.New("test3")},
			expectedTags: []Tag{Unknown, Unknown, Unknown},
			expectedTag:  Unknown,
		},
		{
			description: "all tags",
			errList: []error{
				testTaggedErrors{err: errors.New("test"), tag: 1000},
				testTaggedErrors{err: errors.New("test2"), tag: 2000},
				testTaggedErrors{err: errors.New("test3"), tag: 3000}},
			expectedTags:       []Tag{1000, 2000, 3000},
			expectedTag:        MultipleTags,
			expectedUniqueTags: []Tag{1000, 2000, 3000},
		},
		{
			description: "one tag",
			errList: []error{
				errors.New("test"),
				testTaggedErrors{err: errors.New("test2"), tag: 2000},
				errors.New("test3")},
			expectedTags:       []Tag{Unknown, 2000, Unknown},
			expectedTag:        2000,
			expectedUniqueTags: []Tag{2000},
		},
		{
			description: "multiple same tags",
			errList: []error{
				testTaggedErrors{err: errors.New("test"), tag: 1000},
				testTaggedErrors{err: errors.New("test2"), tag: 2000},
				testTaggedErrors{err: errors.New("test3"), tag: 1000}},
			expectedTags:       []Tag{1000, 2000, 1000},
			expectedTag:        MultipleTags,
			expectedUniqueTags: []Tag{1000, 2000},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			e := Errors(tc.errList)
			for _, err := range tc.errList {
				assert.Contains(e.Error(), err.Error())
				assert.Contains(e.Errors(), err)
				assert.ElementsMatch(tc.expectedTags, e.Tags())
				assert.Equal(tc.expectedTag, e.Tag())
				assert.ElementsMatch(tc.expectedUniqueTags, e.UniqueTags())
			}
		})
	}

}
func TestInvalidEventErr(t *testing.T) {
	testErr := testTaggedErrors{
		err: errors.New("test error"),
		tag: InvalidBirthdate,
	}
	tests := []struct {
		description   string
		underlyingErr error
		tag           Tag
		expectedTag   Tag
	}{
		{
			description: "No underlying error",
		},
		{
			description:   "Underlying error",
			underlyingErr: errors.New("test error"),
		},
		{
			description:   "Underlying error tag",
			underlyingErr: testErr,
			expectedTag:   testErr.Tag(),
		},
		{
			description: "With tag",
			tag:         InvalidBootTime,
			expectedTag: InvalidBootTime,
		},
		{
			description:   "With tag vs underlyingTag",
			tag:           InvalidBootTime,
			underlyingErr: testErr,
			expectedTag:   InvalidBootTime,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			err := InvalidEventErr{OriginalErr: tc.underlyingErr, ErrorTag: tc.tag}
			if tc.underlyingErr != nil {
				assert.Contains(err.Error(), tc.underlyingErr.Error())
			}
			assert.Contains(err.Error(), "event invalid")
			assert.Equal(tc.underlyingErr, err.Unwrap())
			assert.Equal(tc.expectedTag, err.Tag())
		})
	}
}

func TestInvalidBootTimeErr(t *testing.T) {
	tests := []struct {
		description   string
		underlyingErr error
		tag           Tag
	}{
		{
			description: "No underlying error",
		},
		{
			description:   "Underlying error",
			underlyingErr: errors.New("test error"),
		},
		{
			description: "Underlying tag",
			tag:         OldBootTime,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			err := InvalidBootTimeErr{OriginalErr: tc.underlyingErr}
			if tc.tag != Unknown {
				err.ErrorTag = tc.tag
			}
			if tc.underlyingErr != nil {
				assert.Contains(err.Error(), tc.underlyingErr.Error())
			}
			assert.Equal(tc.underlyingErr, err.Unwrap())
			if tc.tag != Unknown {
				assert.Equal(tc.tag, err.Tag())
			} else {
				assert.Equal(InvalidBootTime, err.Tag())
			}
		})
	}
}

func TestInvalidBirthdateErr(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)

	tests := []struct {
		description    string
		underlyingErr  error
		tag            Tag
		timestamps     []int64
		expectedFields []string
	}{
		{
			description: "No underlying error",
		},
		{
			description:   "Underlying error",
			underlyingErr: errors.New("test error"),
		},
		{
			description: "Underlying tag",
			tag:         2000,
		},
		{
			description:   "With fields",
			underlyingErr: errors.New("test error"),
			tag:           2000,
			timestamps:    []int64{now.Unix(), now.Add(time.Hour).Unix(), now.Add(time.Minute).Unix()},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			err := InvalidBirthdateErr{OriginalErr: tc.underlyingErr, ErrorTag: tc.tag, Timestamps: tc.timestamps}
			if tc.underlyingErr != nil {
				assert.Contains(err.Error(), tc.underlyingErr.Error())
			}
			assert.Equal(tc.underlyingErr, err.Unwrap())
			if tc.tag != Unknown {
				assert.Equal(tc.tag, err.Tag())
			} else {
				assert.Equal(InvalidBirthdate, err.Tag())
			}

			var expectedFields []string
			for _, val := range tc.timestamps {
				expectedFields = append(expectedFields, fmt.Sprint(val))
			}
			assert.Equal(expectedFields, err.Fields())
		})
	}
}

func TestInvalidDestinationErr(t *testing.T) {
	tests := []struct {
		description   string
		underlyingErr error
		tag           Tag
	}{
		{
			description: "No underlying error",
		},
		{
			description:   "Underlying error",
			underlyingErr: errors.New("test error"),
		},
		{
			description: "Underlying tag",
			tag:         InvalidEventType,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			err := InvalidDestinationErr{OriginalErr: tc.underlyingErr, ErrorTag: tc.tag}
			if tc.underlyingErr != nil {
				assert.Contains(err.Error(), tc.underlyingErr.Error())
			}
			assert.Equal(tc.underlyingErr, err.Unwrap())
			assert.Contains(err.Error(), "invalid destination")
			if tc.tag != Unknown {
				assert.Equal(tc.tag, err.Tag())
			} else {
				assert.Equal(InvalidDestination, err.Tag())
			}
		})
	}
}

func TestInconsistentIDErr(t *testing.T) {
	tests := []struct {
		description    string
		err            InconsistentIDErr
		expectedFields []string
	}{
		{
			description: "no fields",
			err:         InconsistentIDErr{},
		},
		{
			description:    "with fields",
			err:            InconsistentIDErr{IDs: []string{"test", "test1"}},
			expectedFields: []string{"test", "test1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(InconsistentDeviceID, tc.err.Tag())
			assert.Contains(tc.err.Error(), "inconsistent device id")
			assert.Equal(tc.expectedFields, tc.err.Fields())
		})
	}
}

func TestBootDurationErr(t *testing.T) {
	tests := []struct {
		description   string
		underlyingErr error
		underlyingTag Tag
		expectedTag   Tag
	}{
		{
			description: "No underlying error",
			expectedTag: InvalidBootDuration,
		},
		{
			description:   "Underlying error",
			underlyingErr: errors.New("test error"),
			expectedTag:   InvalidBootDuration,
		},
		{
			description:   "Underlying tag",
			underlyingTag: FastBoot,
			expectedTag:   FastBoot,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			err := BootDurationErr{OriginalErr: tc.underlyingErr, ErrorTag: tc.underlyingTag}
			if tc.underlyingErr != nil {
				assert.Contains(err.Error(), tc.underlyingErr.Error())
			}
			assert.Contains(err.Error(), "boot duration error")
			assert.Equal(tc.underlyingErr, err.Unwrap())
			assert.Equal(tc.expectedTag, err.Tag())
		})
	}
}

func TestEventWithError(t *testing.T) {
	err := errors.New("test")
	tests := []struct {
		description        string
		underlyingErr      error
		event              interpreter.Event
		expectedTag        Tag
		expectedTags       []Tag
		expectedUniqueTags []Tag
	}{
		{
			description:   "with event and err",
			underlyingErr: err,
			event:         interpreter.Event{TransactionUUID: "test"},
			expectedTag:   Unknown,
		},
		{
			description: "tagged err",
			underlyingErr: testTaggedError{
				err: err,
				tag: 2000,
			},
			event:              interpreter.Event{TransactionUUID: "test"},
			expectedTag:        2000,
			expectedTags:       []Tag{2000},
			expectedUniqueTags: []Tag{2000},
		},
		{
			description: "multiple tags",
			underlyingErr: testTaggedErrors{
				err:  err,
				tags: []Tag{2000, 3000, 4000, 3000},
			},
			event:              interpreter.Event{TransactionUUID: "test"},
			expectedTag:        MultipleTags,
			expectedTags:       []Tag{2000, 3000, 4000, 3000},
			expectedUniqueTags: []Tag{2000, 3000, 4000},
		},
		{
			description: "No event",
			underlyingErr: testTaggedErrors{
				err:  err,
				tags: []Tag{2000, 3000, 4000},
			},
			expectedTag:        MultipleTags,
			expectedTags:       []Tag{2000, 3000, 4000},
			expectedUniqueTags: []Tag{2000, 3000, 4000},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			resultingErr := EventWithError{
				Event:       tc.event,
				OriginalErr: tc.underlyingErr,
			}
			assert.Contains(resultingErr.Error(), "event id")
			assert.Equal(tc.underlyingErr, resultingErr.Unwrap())
			assert.Equal(resultingErr.Event, tc.event)
			assert.Equal(tc.expectedTag, resultingErr.Tag())
			assert.Equal(tc.expectedTags, resultingErr.Tags())
			assert.ElementsMatch(tc.expectedUniqueTags, resultingErr.UniqueTags())
		})
	}
}
