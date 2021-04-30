package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			testStr:     "Fast_Boot",
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
