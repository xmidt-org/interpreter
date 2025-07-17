// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
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
	parseCmd.PersistentFlags().BoolVarP(&useRebootParser, "reboot", "r", false, "parse just reboot events")
	rootCmd.AddCommand(parseCmd)
	getEventsCmd.AddCommand(parseCmd)
}

func parse(events []interpreter.Event) {
	cycles := parseIntoCycles(events)
	printBootCycles(cycles)
}

func printBootCycles(cycles []bootCycle) {
	table := tablewriter.NewTable(os.Stdout, tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
		Settings: tw.Settings{
			Separators: tw.Separators{
				BetweenRows: tw.On,
			},
		},
	})))
	table.Configure(func(config *tablewriter.Config) {
		config.Header.Alignment.Global = tw.AlignLeft
		config.Row.Alignment.Global = tw.AlignLeft
		config.Row.Formatting = tw.CellFormatting{
			MergeMode: tw.MergeVertical,
		}
	})
	table.Header([]string{"Cycle ID", "Boot-time", "Birthdate", "Destination", "Event ID"})
	data := make([][]string, 0, len(cycles))
	for _, cycle := range cycles {
		cycleInfo := getCycleInfo(cycle)
		data = append(data, cycleInfo...)
	}

	table.Bulk(data)
	table.Render()
}

func getCycleInfo(cycle bootCycle) [][]string {
	cycleInfo := make([][]string, 0, len(cycle.Events))
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

func parseByParser(events []interpreter.Event, cycleParser history.EventsParserFunc) []bootCycle {
	index := 0
	var cycles []bootCycle
	seenBootTimes := make(map[int64]bool)
	for _, event := range events {
		if boottime, err := event.BootTime(); err == nil && !seenBootTimes[boottime] {
			seenBootTimes[boottime] = true
			parsedEvents, err := cycleParser.Parse(events, event)
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
