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
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/history"
	"github.com/xmidt-org/interpreter/validation"
)

var (
	eventValidator  validation.Validator
	cycleValidators history.CycleValidator
	cycleParser     history.EventsParserFunc
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "validate a list of cycles and events and print",
	PreRun: func(cmd *cobra.Command, args []string) {
		eventValidator, cycleValidators = createValidators()
		cycleParser = history.CurrentCycleParser(nil)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if useRebootParser {
			fmt.Println("Unable to run validation on just reboot events. Parsing by boot-time instead.")
		}
		getEvents(validate)
	},
}

type ValidatorConfig struct {
	Metadata                   []MetadataKeyConfig
	BirthdateAlignmentDuration time.Duration
	MinBootDuration            time.Duration
	ValidEventTypes            []string
	BootTimeValidator          TimeValidationConfig
	BirthdateValidator         TimeValidationConfig
}

type MetadataKeyConfig struct {
	Key              string
	CheckWithinCycle bool
}

type TimeValidationConfig struct {
	ValidFrom    time.Duration
	ValidTo      time.Duration
	MinValidYear int
}

type eventErrs struct {
	event     interpreter.Event
	cycleID   string
	cycleErrs error
	eventErrs error
}

func init() {
	rootCmd.AddCommand(validateCmd)
	parseCmd.AddCommand(validateCmd)
}

func validate(events []interpreter.Event) {
	cycles := parseByParser(events, cycleParser)
	var allErrors []eventErrs
	for _, cycle := range cycles {
		_, cycleErrs := cycleValidators.Valid(cycle.Events)
		for _, event := range cycle.Events {
			_, err := eventValidator.Valid(event)
			allErrors = append(allErrors, eventErrs{
				event:     event,
				cycleID:   cycle.ID,
				cycleErrs: cycleErrs,
				eventErrs: err,
			})
		}
	}

	printValidationTable(allErrors)
}

func printValidationTable(info []eventErrs) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{"Cycle", "Event ID", "Boot-time", "Destination", "Event Errors", "Cycle Errors"})
	data := make([][]string, 0, len(info))
	for _, eventErr := range info {
		data = append(data, getValidationRowInfo(eventErr))
	}
	table.SetAutoMergeCellsByColumnIndex([]int{0, 2, 5})
	table.SetRowLine(true)
	table.AppendBulk(data)
	table.Render()
}

func getValidationRowInfo(info eventErrs) []string {
	return []string{
		info.cycleID,
		info.event.TransactionUUID,
		getBoottimeString(info.event),
		info.event.Destination,
		errorTagsToString(info.eventErrs),
		errorTagsToString(info.cycleErrs),
	}
}

func createValidators() (validation.Validator, history.CycleValidator) {
	var config ValidatorConfig
	viper.UnmarshalKey("validators", &config)
	cycleValidators := createCycleValidators(config)
	eventValidator := createEventValidators(config)
	return eventValidator, cycleValidators
}

func createCycleValidators(config ValidatorConfig) history.CycleValidator {
	validators := []history.CycleValidator{
		history.TransactionUUIDValidator(),
		history.SessionOnlineValidator(func(_ []interpreter.Event, _ string) bool { return false }),
		history.SessionOfflineValidator(func(_ []interpreter.Event, _ string) bool { return false }),
	}
	var withinCycleChecks []string
	var wholeCycleChecks []string
	for _, metadata := range config.Metadata {
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

	return history.CycleValidators(validators)
}

func createEventValidators(config ValidatorConfig) validation.Validator {
	bootTimeValidator := validation.BootTimeValidator(validation.TimeValidator{
		Current:      time.Now,
		ValidFrom:    config.BootTimeValidator.ValidFrom,
		ValidTo:      config.BootTimeValidator.ValidTo,
		MinValidYear: config.BootTimeValidator.MinValidYear,
	})

	birthdateValidator := validation.BirthdateValidator(validation.TimeValidator{
		Current:      time.Now,
		ValidFrom:    config.BirthdateValidator.ValidFrom,
		ValidTo:      config.BirthdateValidator.ValidTo,
		MinValidYear: config.BirthdateValidator.MinValidYear,
	})

	birthdateAlignmentValidator := validation.BirthdateAlignmentValidator(config.BirthdateAlignmentDuration)
	consistentIDValidator := validation.ConsistentDeviceIDValidator()
	bootDurationValidator := validation.BootDurationValidator(config.MinBootDuration)
	eventTypeValidator := validation.EventTypeValidator(config.ValidEventTypes)

	validators := validation.Validators([]validation.Validator{
		bootTimeValidator, birthdateValidator, birthdateAlignmentValidator, consistentIDValidator, bootDurationValidator, eventTypeValidator,
	})

	return validators
}
