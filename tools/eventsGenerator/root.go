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
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xmidt-org/interpreter"
)

var (
	cfgFile         string
	destinationFile string

	rootCmd = &cobra.Command{
		Use:   "eventsGenerator",
		Short: "eventsGenerator generates a json containing a list of events from a yaml config file",
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
)

type Config struct {
	MessageContents []Message
}

type Message struct {
	Event           interpreter.Event
	Payload         map[string]string
	BootTimeOffset  time.Duration
	BirthdateOffset time.Duration
}

func init() {
	cobra.OnInitialize(initializePaths)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./eventsGenerator.yaml)")
	rootCmd.PersistentFlags().StringVarP(&destinationFile, "destination", "d", "", "destination for resulting json that contains list of events (default is ./events.json)")
}

func initializePaths() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("eventsGenerator")
	}

	if len(destinationFile) == 0 {
		destinationFile = "./events.json"
	}

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func generateEvents(messages []Message) []interpreter.Event {
	now := time.Now()
	events := make([]interpreter.Event, 0, len(messages))
	for i, msg := range messages {
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

func writeEvents(events []interpreter.Event) error {
	if data, err := json.Marshal(events); err == nil {
		writeErr := ioutil.WriteFile(destinationFile, data, 0644) // nolint:gosec
		if writeErr != nil {
			return writeErr
		}
	}

	return nil
}

func run() {
	var messages []Message
	viper.UnmarshalKey("messageContents", &messages)
	events := generateEvents(messages)
	if err := writeEvents(events); err == nil {
		os.Exit(0)
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
