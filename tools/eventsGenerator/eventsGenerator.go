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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/interpreter"
	"go.uber.org/fx"
)

const (
	applicationName = "eventsGenerator"
)

type Config struct {
	MessageContents []Message
	FilePath        string
}

type Message struct {
	Event           interpreter.Event
	Payload         map[string]string
	BootTimeOffset  time.Duration
	BirthdateOffset time.Duration
}

func generateEvents(config Config) []interpreter.Event {
	now := time.Now()
	events := make([]interpreter.Event, 0, len(config.MessageContents))
	for i, msg := range config.MessageContents {
		if len(msg.Event.TransactionUUID) == 0 {
			msg.Event.TransactionUUID = strconv.Itoa(i)
		}
		events = append(events, createEvent(now, msg))
	}

	sort.Slice(events, func(a int, b int) bool {
		return events[a].Birthdate > events[b].Birthdate
	})
	return events
}

func createEvent(current time.Time, msg Message) interpreter.Event {
	event := msg.Event
	event.MsgType = 4
	event.Metadata = make(map[string]string)

	for k, v := range msg.Event.Metadata {
		event.Metadata[k] = v
	}

	payload := make(map[string]string)
	for k, v := range msg.Payload {
		payload[k] = v
	}

	event.Metadata["/boot-time"] = fmt.Sprint(current.Add(msg.BootTimeOffset).Unix())
	birthdate := current.Add(msg.BirthdateOffset)
	payload["ts"] = current.Add(msg.BirthdateOffset).Format(time.RFC3339Nano)
	if j, err := json.Marshal(payload); err == nil {
		event.Payload = string(j)
	} else {
		event.Payload = fmt.Sprintf(`{"ts":"%s"}`, birthdate)
	}
	event.Birthdate = birthdate.UnixNano()
	return event
}

func writeEvents(filePath string, events []interpreter.Event) {
	if data, err := json.Marshal(events); err == nil {
		writeErr := ioutil.WriteFile(filePath, data, 0644) // nolint:gosec
		if writeErr != nil {
			panic(writeErr)
		}
	}
}

func main() {
	var configFile string
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	} else {
		configFile = fmt.Sprintf("./%s.yaml", applicationName)
	}

	v := viper.New()
	v.SetConfigFile(configFile)
	err := v.ReadInConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read in viper config: %v\n", err.Error())
		os.Exit(1)
	}

	app := fx.New(
		arrange.ForViper(v),
		arrange.Provide(Config{}),
		fx.Provide(
			generateEvents,
		),
		fx.Invoke(
			func(config Config, events []interpreter.Event) {
				var filePath string
				if len(os.Args) > 2 {
					filePath = os.Args[2]
				} else {
					filePath = config.FilePath
				}

				writeEvents(filePath, events)
				os.Exit(0)
			},
		),
	)

	if err := app.Err(); err == nil {
		app.Run()
	} else if errors.Is(err, pflag.ErrHelp) {
		return
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
