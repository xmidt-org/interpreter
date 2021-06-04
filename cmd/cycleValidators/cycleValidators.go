package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xmidt-org/arrange"
	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/history"
	"go.uber.org/fx"
)

const (
	applicationName = "cycleValidators"
)

type Config struct {
	FilePath           string
	MetadataValidators []MetadataKey
}

type MetadataKey struct {
	Key              string
	CheckWithinCycle bool
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

func readCommandLine(config Config) {
	validators := createValidators(config)
	var events []interpreter.Event
	var filePath string
	if len(os.Args) > 1 {
		filePath = os.Args[1]
	} else {
		filePath = config.FilePath
	}
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to read from file")
		os.Exit(1)
	}

	if err = json.Unmarshal(data, &events); err != nil {
		fmt.Fprintf(os.Stderr, "unable to unmarshal json: %v", err)
		os.Exit(1)
	}

	errs := runValidators(events, validators)
	printErrors(errs)
	os.Exit(0)
}

func createValidators(config Config) []history.CycleValidatorFunc {
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

func runValidators(events []interpreter.Event, validators []history.CycleValidatorFunc) []error {
	var allErr []error
	for _, validator := range validators {
		if valid, err := validator.Valid(events); !valid {
			allErr = append(allErr, err)
		}
	}

	if len(allErr) == 0 {
		return nil
	}

	return allErr
}

func printErrors(errs []error) {
	for _, err := range errs {
		var cvErr history.CycleValidationErr
		if errors.As(err, &cvErr) {
			fmt.Fprintf(os.Stdout, "Tag: %v; Fields: %v; Error: %v\n", cvErr.Tag(), cvErr.Fields(), cvErr)
		} else {
			fmt.Fprintln(os.Stdout, "error: ", err)
		}
	}
}
