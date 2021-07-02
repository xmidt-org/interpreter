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

var ParseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse list of events into cycles and print",
	PreRun: func(cmd *cobra.Command, args []string) {
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

type BootCycle struct {
	ID     string
	Events []interpreter.Event
	Err    error
}

func init() {
	GetEventsCmd.AddCommand(ParseCmd)
	RootCmd.AddCommand(ParseCmd)
}

func parse(events []interpreter.Event) {
	cycles := parseIntoCycles(events)
	printBootCycles(cycles)
}

func printBootCycles(cycles []BootCycle) {
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

func getCycleInfo(cycle BootCycle) [][]string {
	var cycleInfo [][]string
	for _, event := range cycle.Events {
		eventInfo := []string{cycle.ID, getBoottimeString(event), getBirthdateString(event), event.Destination, event.TransactionUUID}
		cycleInfo = append(cycleInfo, eventInfo)
	}
	return cycleInfo
}

func parseIntoCycles(events []interpreter.Event) []BootCycle {
	index := 0
	var cycles []BootCycle
	seenBootTimes := make(map[int64]bool)
	for _, event := range events {
		if boottime, err := event.BootTime(); err == nil && !seenBootTimes[boottime] {
			seenBootTimes[boottime] = true
			parsedEvents, err := parser.Parse(events, event)
			cycles = append(cycles, BootCycle{
				ID:     strconv.Itoa(index),
				Events: parsedEvents,
				Err:    err,
			})
			index++
		}
	}

	return cycles
}
