package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/history"
	"github.com/xmidt-org/interpreter/validation"
	"go.uber.org/fx"
)

const (
	applicationName = "eventsParser"
)

type Config struct {
	Codex    CodexConfig
	FilePath string
	UseJSON  bool
}

type BootCycle struct {
	ID       string
	EventIDs []string
	Err      error
}

func main() {
	v := viper.New()
	v.AddConfigPath(".")
	v.SetConfigName(applicationName)
	err := v.ReadInConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read in viper config: %v\n", err.Error())
		os.Exit(1)
	}

	app := fx.New(
		arrange.ForViper(v),
		arrange.Provide(Config{}),
		Provide(),
		fx.Provide(
			arrange.UnmarshalKey("codex", CodexConfig{}),
		),
		fx.Invoke(
			readCommandLine,
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

func readCommandLine(config Config, client *CodexClient) {
	validators := createValidators()
	comparator := testComparator()
	if config.UseJSON {
		var events []interpreter.Event
		data, err := ioutil.ReadFile(config.FilePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to read from file")
			os.Exit(1)
		}

		if err = json.Unmarshal(data, &events); err != nil {
			fmt.Fprintf(os.Stderr, "unable to unmarshal json: %v", err)
			os.Exit(1)
		}

		bootCycles := parseIntoCycles(events, comparator, validators)
		printBootCycles(bootCycles)
		os.Exit(0)
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for {
			scanner.Scan()
			id := scanner.Text()
			if len(id) > 0 {
				events := client.getEvents(id)
				bootCycles := parseIntoCycles(events, comparator, validators)
				printBootCycles(bootCycles)
			}
		}
	}
}

func parseIntoCycles(events []interpreter.Event, comparator history.Comparator, validator validation.Validator) []BootCycle {
	index := 0
	var cycles []BootCycle
	parser := history.BootCycleParser(comparator, validator)
	seenBootTimes := make(map[int64]bool)
	for _, event := range events {
		if boottime, err := event.BootTime(); err == nil && seenBootTimes[boottime] != true {
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

func testComparator() history.ComparatorFunc {
	return func(interpreter.Event, interpreter.Event) (bool, error) {
		return false, nil
	}
}

func createValidators() validation.Validator {
	bootTimeValidator := validation.BootTimeValidator(validation.TimeValidator{
		Current:      time.Now,
		ValidFrom:    -8766 * time.Hour, // 1 year
		ValidTo:      time.Hour,
		MinValidYear: 2015,
	})

	birthdateValidator := validation.BirthdateValidator(validation.TimeValidator{
		Current:      time.Now,
		ValidFrom:    -8766 * time.Hour, // 1 year
		ValidTo:      time.Hour,
		MinValidYear: 2015,
	})

	birthdateAlignmentValidator := validation.BirthdateAlignmentValidator(time.Hour)
	consistentIDValidator := validation.ConsistentDeviceIDValidator()
	bootDurationValidator := validation.BootDurationValidator(10 * time.Second)
	eventTypeValidator := validation.EventTypeValidator()

	validators := validation.Validators([]validation.Validator{
		bootTimeValidator, birthdateValidator, birthdateAlignmentValidator, consistentIDValidator, bootDurationValidator, eventTypeValidator,
	})

	return validators
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
		fmt.Fprint(os.Stdout, err)
		return
	}

	for _, err := range allErrors {
		var eventWithErr validation.EventWithError
		if errors.As(err, &eventWithErr) {
			fmt.Fprintf(os.Stdout, "Event %s: %v\n", eventWithErr.Event.TransactionUUID, eventWithErr.Tags())
		} else {
			fmt.Fprintf(os.Stdout, "error: %v\n", err)
		}
	}
}
