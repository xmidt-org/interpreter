package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidEventErr(t *testing.T) {
	testErr := testError{
		err:   errors.New("test error"),
		label: "test error",
	}
	tests := []struct {
		description   string
		underlyingErr error
		expectedLabel string
		tag           Tag
	}{
		{
			description:   "No underlying error",
			expectedLabel: invalidEventReason,
		},
		{
			description:   "Underlying error",
			underlyingErr: errors.New("test error"),
			expectedLabel: invalidEventReason,
		},
		{
			description:   "Underlying error label",
			underlyingErr: testErr,
			expectedLabel: "test_error",
		},
		{
			description:   "With tag",
			tag:           InvalidBootTime,
			expectedLabel: invalidEventReason,
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
			assert.Equal(tc.expectedLabel, err.ErrorLabel())
			assert.Equal(tc.tag, err.Tag())
		})
	}
}

func TestInvalidBootTimeErr(t *testing.T) {
	tests := []struct {
		description   string
		underlyingErr error
		expectedLabel string
		tag           Tag
	}{
		{
			description:   "No underlying error",
			expectedLabel: invalidBootTimeReason,
		},
		{
			description:   "Underlying error",
			underlyingErr: errors.New("test error"),
			expectedLabel: invalidBootTimeReason,
		},
		{
			description:   "Underlying tag",
			expectedLabel: invalidBootTimeReason,
			tag:           OldBootTime,
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
			assert.Equal(tc.expectedLabel, err.ErrorLabel())
			if tc.tag != Unknown {
				assert.Equal(tc.tag, err.Tag())
			} else {
				assert.Equal(InvalidBootTime, err.Tag())
			}
		})
	}
}

func TestInvalidBirthdateErr(t *testing.T) {
	tests := []struct {
		description   string
		underlyingErr error
		expectedLabel string
	}{
		{
			description:   "No underlying error",
			expectedLabel: invalidBirthdateReason,
		},
		{
			description:   "Underlying error",
			underlyingErr: errors.New("test error"),
			expectedLabel: invalidBirthdateReason,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			err := InvalidBirthdateErr{OriginalErr: tc.underlyingErr}
			if tc.underlyingErr != nil {
				assert.Contains(err.Error(), tc.underlyingErr.Error())
			}
			assert.Equal(tc.underlyingErr, err.Unwrap())
			assert.Equal(tc.expectedLabel, err.ErrorLabel())
		})
	}
}

func TestInvalidDestinationErr(t *testing.T) {
	tests := []struct {
		description   string
		underlyingErr error
		label         string
		expectedLabel string
	}{
		{
			description:   "No underlying error",
			expectedLabel: invalidDestinationReason,
		},
		{
			description:   "Underlying error",
			underlyingErr: errors.New("test error"),
			expectedLabel: invalidDestinationReason,
		},
		{
			description:   "Underlying error with label",
			underlyingErr: errors.New("test error"),
			label:         "test_error",
			expectedLabel: "test_error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			err := InvalidDestinationErr{OriginalErr: tc.underlyingErr, ErrLabel: tc.label}
			if tc.underlyingErr != nil {
				assert.Contains(err.Error(), tc.underlyingErr.Error())
			}
			assert.Equal(tc.underlyingErr, err.Unwrap())
			assert.Contains(err.Error(), "invalid destination")
			assert.Equal(tc.expectedLabel, err.ErrorLabel())
		})
	}
}
