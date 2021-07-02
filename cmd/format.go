package main

import (
	"errors"
	"strings"
	"time"

	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

const (
	timeFormat = "Jan 2 15:04:05.00000"
)

func getBoottimeString(event interpreter.Event) string {
	boottime, err := event.BootTime()
	if err != nil || boottime <= 0 {
		return "error"
	}
	return time.Unix(boottime, 0).UTC().Format(timeFormat)
}

func getBirthdateString(event interpreter.Event) string {
	return time.Unix(0, event.Birthdate).UTC().Format(timeFormat)
}

func errorTagsToString(err error) string {
	if err == nil {
		return ""
	}

	var taggedErr validation.TaggedError
	var taggedErrs validation.TaggedErrors
	var tags []validation.Tag
	if errors.As(err, &taggedErrs) {
		tags = taggedErrs.UniqueTags()
	} else if errors.As(err, &taggedErr) {
		tags = []validation.Tag{taggedErr.Tag()}
	} else {
		return err.Error()
	}

	var output strings.Builder
	for i, tag := range tags {
		if i > 0 {
			output.WriteRune('\n')
		}
		output.WriteString(tag.String())
	}

	return output.String()
}
