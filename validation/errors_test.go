package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidEventErr(t *testing.T) {
	testErr := testError{
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
			tag:         MisalignedBirthdate,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			err := InvalidBirthdateErr{OriginalErr: tc.underlyingErr, ErrorTag: tc.tag}
			if tc.underlyingErr != nil {
				assert.Contains(err.Error(), tc.underlyingErr.Error())
			}
			assert.Equal(tc.underlyingErr, err.Unwrap())
			if tc.tag != Unknown {
				assert.Equal(tc.tag, err.Tag())
			} else {
				assert.Equal(InvalidBirthdate, err.Tag())
			}
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
	err := InconsistentIDErr{}
	assert.Equal(t, InconsistentDeviceID, err.Tag())
	assert.Contains(t, err.Error(), "inconsistent device id")
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
			assert.Equal(tc.expectedTag, err.Tag())
		})
	}
}
