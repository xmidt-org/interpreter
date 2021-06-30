package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/history"
	"github.com/xmidt-org/interpreter/validation"
)

var ParseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Parse list of events into cycles and print",
	Run: func(cmd *cobra.Command, args []string) {
		if len(eventsFile) > 0 {
			readCommandLine(eventsFile)
		} else {
			// TODO: fill in
			// scanner := bufio.NewScanner(os.Stdin)
			// for scanner.Scan() {
			// 	id := scanner.Text()
			// 	if len(id) > 0 {
			// 		events := client.getEvents(id)
			// 		bootCycles := parseIntoCycles(events, nil, nil)
			// 		printBootCycles(bootCycles)
			// 	}
			// }

			// if err := scanner.Err(); err != nil {
			// 	fmt.Fprintln(os.Stderr, "reading standard input:", err)
			// }

			fmt.Println("no file name")
		}

	},
}

type BootCycle struct {
	ID       string
	EventIDs []string
	Err      error
}

func init() {
	RootCmd.AddCommand(ParseCmd)
}

func readCommandLine(filePath string) {
	var events []interpreter.Event
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to read from file")
		os.Exit(1)
	}

	if err = json.Unmarshal(data, &events); err != nil {
		fmt.Fprintf(os.Stderr, "unable to unmarshal json: %v", err)
		os.Exit(1)
	}

	bootCycles := parseIntoCycles(events, nil, nil)
	printBootCycles(bootCycles)
	os.Exit(0)
}

func parseIntoCycles(events []interpreter.Event, comparator history.Comparator, validator validation.Validator) []BootCycle {
	index := 0
	var cycles []BootCycle
	parser := history.BootCycleParser(comparator, validator)
	seenBootTimes := make(map[int64]bool)
	for _, event := range events {
		if boottime, err := event.BootTime(); err == nil && !seenBootTimes[boottime] {
			seenBootTimes[boottime] = true
			parsedEvents, err := parser.Parse(events, event)
			var ids []string
			for _, parsedEvent := range parsedEvents {
				ids = append(ids, parsedEvent.TransactionUUID)
			}
			cycles = append(cycles, BootCycle{
				ID:       strconv.Itoa(index),
				EventIDs: ids,
				Err:      err,
			})
			index++
		}
	}

	return cycles
}

func printBootCycles(cycles []BootCycle) {
	for _, cycle := range cycles {
		fmt.Fprintf(os.Stdout, "--------CYCLE ID %s----------\n", cycle.ID)
		fmt.Fprintf(os.Stdout, "Event IDs: %v\n", cycle.EventIDs)
		fmt.Fprintln(os.Stdout, "Errors:")
		printErrorTags(cycle.Err)
	}
}

func printErrorTags(err error) {
	var allErrors validation.Errors
	if !errors.As(err, &allErrors) {
		fmt.Fprintln(os.Stdout, err)
		return
	}

	for _, err := range allErrors {
		var eventWithErr validation.EventWithError
		if errors.As(err, &eventWithErr) {
			fmt.Fprintf(os.Stdout, "Event %s: %v\n", eventWithErr.Event.TransactionUUID, eventWithErr.Tags())
		} else {
			fmt.Fprintln(os.Stdout, "error: ", err)
		}
	}
}
