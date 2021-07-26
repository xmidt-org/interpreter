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

	var nonExistentTag Tag = 1000
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

func TestTagsToStrings(t *testing.T) {
	tests := []struct {
		description     string
		tags            []Tag
		expectedStrings []string
	}{
		{
			description:     "multiple tags",
			tags:            []Tag{RepeatedTransactionUUID, Unknown, DuplicateEvent, Tag(2000)},
			expectedStrings: []string{RepeatedTransactionUUIDStr, UnknownStr, DuplicateEventStr, UnknownStr},
		},
		{
			description:     "one tag",
			tags:            []Tag{RepeatedTransactionUUID},
			expectedStrings: []string{RepeatedTransactionUUIDStr},
		},
		{
			description:     "empty list",
			tags:            []Tag{},
			expectedStrings: []string{},
		},
		{
			description:     "nil list",
			expectedStrings: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			str := TagsToStrings(tc.tags)
			assert.ElementsMatch(tc.expectedStrings, str)
		})
	}
}
