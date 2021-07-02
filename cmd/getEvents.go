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
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xmidt-org/interpreter"
)

var getEventsCmd = &cobra.Command{
	Use:   "get",
	Short: "Gets and prints list of events",
	Run: func(cmd *cobra.Command, args []string) {
		getEvents(printEvents)
	},
}

func init() {
	rootCmd.AddCommand(getEventsCmd)
}

func getEvents(eventsCallback func([]interpreter.Event)) {
	if len(eventsFile) > 0 {
		events, err := readFile(eventsFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		eventsCallback(events)
		os.Exit(0)
	} else {
		var config CodexConfig
		viper.UnmarshalKey("codex", &config)
		auth, _ := createCodexAuth(config)
		client := createClient(config, auth)
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			id := scanner.Text()
			if len(id) > 0 {
				events := client.getEvents(id)
				eventsCallback(events)
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}

		fmt.Println("no file name")
	}
}

func readFile(filePath string) ([]interpreter.Event, error) {
	var events []interpreter.Event
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return events, fmt.Errorf("unable to read from file: %v", err)
	}

	if err = json.Unmarshal(data, &events); err != nil {
		return events, fmt.Errorf("unable to unmarshal json: %v", err)
	}

	return events, nil
}

func printEvents(events []interpreter.Event) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{"ID", "Boot-time", "Birthdate", "Destination"})
	var data [][]string
	for _, event := range events {
		data = append(data, getEventInfo(event))
	}
	table.SetAutoMergeCells(true)
	table.AppendBulk(data)
	table.Render()
}

func getEventInfo(event interpreter.Event) []string {
	return []string{event.TransactionUUID, getBoottimeString(event), getBirthdateString(event), event.Destination}
}
