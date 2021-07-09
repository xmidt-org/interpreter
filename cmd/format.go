/**
 * Copyright 2021 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package main

import (
	"errors"
	"strings"
	"time"

	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
)

const (
	timeFormat = "Jan 2 2006 15:04:05.00000"
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
