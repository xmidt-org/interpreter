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

package interpreter

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xmidt-org/wrp-go/v3"
)

const (
	BootTimeKey = "/boot-time"
)

var (
	ErrParseDeviceID    = errors.New("error getting device ID from event")
	ErrBirthdateParse   = errors.New("unable to parse birthdate from payload")
	ErrBootTimeParse    = errors.New("unable to parse boot-time")
	ErrBootTimeNotFound = errors.New("boot-time not found")
	ErrTypeNotFound     = errors.New("type not found")

	// EventRegex is the regex that an event's destination must match in order to parse the device id properly.
	EventRegex = regexp.MustCompile(`^(?P<event>[^/]+)/((?P<prefix>(?i)mac|uuid|dns|serial):(?P<id>[^/]+))/(?P<type>[^/\s]+)`)

	// DeviceIDRegex is used to parse a device id from anywhere.
	DeviceIDRegex = regexp.MustCompile(`(?P<prefix>(?i)mac|uuid|dns|serial):(?P<id>[^/]+)`)
)

// Event is the struct that contains the wrp.Message fields along with the birthdate
// that is parsed from the payload.
type Event struct {
	MsgType         int               `json:"msg_type"`
	Source          string            `json:"source"`
	Destination     string            `json:"dest,omitempty"`
	TransactionUUID string            `json:"transaction_uuid,omitempty"`
	ContentType     string            `json:"content_type,omitempty"`
	Metadata        map[string]string `json:"metadata"`
	Payload         string            `json:"payload,omitempty"`
	Birthdate       int64             `json:"birth_date"`
	PartnerIDs      []string          `json:"partner_ids,omitempty"`
}

// NewEvent creates an Event from a wrp.Message and also parses the Birthdate from the
// message payload. A new Event will always be returned from this function, but if the
// birthdate cannot be parsed from the payload, it will return an error along with the Event created.
func NewEvent(msg wrp.Message) (Event, error) {
	var err error
	event := Event{
		MsgType:         int(msg.MessageType()),
		Source:          msg.Source,
		Destination:     msg.Destination,
		TransactionUUID: msg.TransactionUUID,
		ContentType:     msg.ContentType,
		Metadata:        msg.Metadata,
		Payload:         string(msg.Payload),
		PartnerIDs:      msg.PartnerIDs,
	}

	if birthdate, ok := getBirthDate(msg.Payload); ok {
		event.Birthdate = birthdate.UnixNano()
	} else {
		err = ErrBirthdateParse
	}

	return event, err
}

// GetMetadataValue checks the metadata map for a specific key,
// allowing for keys with or without forward-slash.
func (e Event) GetMetadataValue(key string) (string, bool) {
	value, found := e.Metadata[key]
	if !found {
		value, found = e.Metadata[strings.Trim(key, "/")]
	}

	return value, found
}

// BootTime parses the boot-time from an event, returning an
// error if the boot-time doesn't exist or cannot be parsed.
func (e Event) BootTime() (int64, error) {
	bootTimeStr, ok := e.GetMetadataValue(BootTimeKey)
	if !ok {
		return 0, ErrBootTimeNotFound
	}

	bootTime, err := strconv.ParseInt(bootTimeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrBootTimeParse, err)
	}

	return bootTime, err
}

// DeviceID gets the device id based on the event regex.
func (e Event) DeviceID() (string, error) {
	match := EventRegex.FindStringSubmatch(e.Destination)
	if len(match) < 3 {
		return "", ErrParseDeviceID
	}

	return match[2], nil
}

// EventType returns the event type from the event's destination.
func (e Event) EventType() (string, error) {
	match := EventRegex.FindStringSubmatch(e.Destination)
	if len(match) < 6 {
		return "", ErrTypeNotFound
	}

	return match[5], nil
}

func getBirthDate(payload []byte) (time.Time, bool) {
	p := make(map[string]interface{})
	if len(payload) == 0 {
		return time.Time{}, false
	}
	err := json.Unmarshal(payload, &p)
	if err != nil {
		return time.Time{}, false
	}

	// parse the time from the payload
	timeString, ok := p["ts"].(string)
	if !ok {
		return time.Time{}, false
	}
	birthDate, err := time.Parse(time.RFC3339Nano, timeString)
	if err != nil {
		return time.Time{}, false
	}
	return birthDate, true
}
