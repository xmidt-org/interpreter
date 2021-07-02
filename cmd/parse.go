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
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/history"
)

var parser history.EventsParserFunc

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse list of events into cycles and print",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if useRebootParser {
			parser = history.RebootParser(nil)
		} else {
			parser = history.CurrentCycleParser(nil)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		getEvents(parse)
	},
}

type bootCycle struct {
	ID     string
	Events []interpreter.Event
	Err    error
}

func init() {
	rootCmd.AddCommand(parseCmd)
	getEventsCmd.AddCommand(parseCmd)
}

func parse(events []interpreter.Event) {
	cycles := parseIntoCycles(events)
	printBootCycles(cycles)
}

func printBootCycles(cycles []bootCycle) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{"Cycle ID", "Boot-time", "Birthdate", "Destination", "ID"})
	var data [][]string
	for _, cycle := range cycles {
		cycleInfo := getCycleInfo(cycle)
		for _, eventInfo := range cycleInfo {
			data = append(data, eventInfo)
		}
	}

	mergeColumns := []int{0}
	if !useRebootParser {
		mergeColumns = []int{0, 1}
	}
	table.SetAutoMergeCellsByColumnIndex(mergeColumns)
	table.SetRowLine(true)
	table.AppendBulk(data)
	table.Render()
}

func getCycleInfo(cycle bootCycle) [][]string {
	var cycleInfo [][]string
	for _, event := range cycle.Events {
		eventInfo := []string{cycle.ID, getBoottimeString(event), getBirthdateString(event), event.Destination, event.TransactionUUID}
		cycleInfo = append(cycleInfo, eventInfo)
	}
	return cycleInfo
}

func parseIntoCycles(events []interpreter.Event) []bootCycle {
	index := 0
	var cycles []bootCycle
	seenBootTimes := make(map[int64]bool)
	for _, event := range events {
		if boottime, err := event.BootTime(); err == nil && !seenBootTimes[boottime] {
			seenBootTimes[boottime] = true
			parsedEvents, err := parser.Parse(events, event)
			cycles = append(cycles, bootCycle{
				ID:     strconv.Itoa(index),
				Events: parsedEvents,
				Err:    err,
			})
			index++
		}
	}

	return cycles
}
