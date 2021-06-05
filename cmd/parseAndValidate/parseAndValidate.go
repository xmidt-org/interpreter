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
	applicationName = "parseAndValidate"
)

type Config struct {
	Codex              CodexConfig
	FilePath           string
	MetadataValidators []MetadataKey
	UseJSON            bool
}

type MetadataKey struct {
	Key              string
	CheckWithinCycle bool
}

type BootCycle struct {
	ID                  string
	Events              []interpreter.Event
	EventIDs            []string
	IndividualEventErrs error
	CycleErrs           error
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
	eventValidators := createEventValidators()
	cycleValidators := createCycleValidators(config)
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
		bootCycles := parseIntoCycles(events, comparator, eventValidators)
		bootCycles = runCycleValidators(bootCycles, cycleValidators)
		printBootCycles(bootCycles)
		os.Exit(0)
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			id := scanner.Text()
			if len(id) > 0 {
				events := client.getEvents(id)
				bootCycles := parseIntoCycles(events, comparator, eventValidators)
				bootCycles = runCycleValidators(bootCycles, cycleValidators)
				printBootCycles(bootCycles)
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
	}
}

func runCycleValidators(cycles []BootCycle, cycleValidators []history.CycleValidatorFunc) []BootCycle {
	for i, cycle := range cycles {
		var allErrs validation.Errors
		for _, validator := range cycleValidators {
			if valid, err := validator.Valid(cycle.Events); !valid {
				allErrs = append(allErrs, err)
			}
		}

		if len(allErrs) > 0 {
			cycle.CycleErrs = allErrs
			cycles[i] = cycle
		}
	}

	return cycles
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
				ID:                  strconv.Itoa(index),
				Events:              parsedEvents,
				EventIDs:            ids,
				IndividualEventErrs: err,
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

func createCycleValidators(config Config) []history.CycleValidatorFunc {
	validators := []history.CycleValidatorFunc{
		history.TransactionUUIDValidator(),
		history.SessionOnlineValidator(func(id string) bool { return false }),
		history.SessionOfflineValidator(func(id string) bool { return false }),
	}
	var withinCycleChecks []string
	var wholeCycleChecks []string
	for _, metadata := range config.MetadataValidators {
		if metadata.CheckWithinCycle {
			withinCycleChecks = append(withinCycleChecks, metadata.Key)
		} else {
			wholeCycleChecks = append(wholeCycleChecks, metadata.Key)
		}
	}

	if len(withinCycleChecks) > 0 {
		validators = append(validators, history.MetadataValidator(withinCycleChecks, true))
	}

	if len(wholeCycleChecks) > 0 {
		validators = append(validators, history.MetadataValidator(wholeCycleChecks, false))
	}

	return validators
}

func createEventValidators() validation.Validator {
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
		fmt.Fprintf(os.Stdout, "************CYCLE ID %s************\n", cycle.ID)
		fmt.Fprintf(os.Stdout, "Event IDs: %v\n", cycle.EventIDs)
		fmt.Fprintln(os.Stdout, "---Individual Errors---")
		printIndividualEventErrorTags(cycle.IndividualEventErrs)
		fmt.Fprintln(os.Stdout, "---Cycle Errors---")
		printCycleErrorTags(cycle.CycleErrs)
	}
}

func printCycleErrorTags(err error) {
	if err == nil {
		fmt.Println("nil")
		return
	}

	var allErrors validation.Errors
	if !errors.As(err, &allErrors) {
		fmt.Fprint(os.Stdout, err)
		return
	}

	for _, err := range allErrors {
		var cvErr history.CycleValidationErr
		if errors.As(err, &cvErr) {
			fmt.Fprintln(os.Stdout, cvErr.Tag())
		} else {
			fmt.Fprintln(os.Stdout, "error: ", err)
		}
	}
}

func printIndividualEventErrorTags(err error) {
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
			fmt.Fprintln(os.Stdout, "error: ", err)
		}
	}
}
