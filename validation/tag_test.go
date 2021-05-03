package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	tests := []Tag{Pass, Unknown, InvalidBootTime, OldBootTime, InconsistentDeviceID, MissingBootTime, FastBoot}
	for _, tag := range tests {
		assert.Equal(t, tagToString[tag], tag.String())
	}

	var nonExistentTag Tag
	nonExistentTag = 1000
	assert.Equal(t, "unknown", nonExistentTag.String())
}
func TestParseTag(t *testing.T) {
	tests := []struct {
		testStr     string
		expectedTag Tag
	}{
		{
			testStr:     "pass",
			expectedTag: Pass,
		},
		{
			testStr:     "random",
			expectedTag: Unknown,
		},
		{
			testStr:     "Suspiciously_Fast_Boot",
			expectedTag: FastBoot,
		},
		{
			testStr:     "Inconsistent device id",
			expectedTag: InconsistentDeviceID,
		},
	}

	for _, tc := range tests {
		t.Run(tc.testStr, func(t *testing.T) {
			assert := assert.New(t)
			tag := ParseTag(tc.testStr)
			assert.Equal(tc.expectedTag, tag)
		})
	}
}
