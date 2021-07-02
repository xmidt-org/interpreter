package main

import (
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
	comparator      history.Comparator
)

var ValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "validate a list of cycles and events and print",
	PreRun: func(cmd *cobra.Command, args []string) {
		eventValidator = createEventValidators()
		cycleValidators = createCycleValidators()
		comparator = createComparator()
	},
	Run: func(cmd *cobra.Command, args []string) {
		getEvents(validate)
	},
}

type MetadataKey struct {
	Key              string
	CheckWithinCycle bool
}

type EventErrs struct {
	event     interpreter.Event
	cycleID   string
	cycleErrs error
	eventErrs error
}

func init() {
	ParseCmd.AddCommand(ValidateCmd)
	RootCmd.AddCommand(ValidateCmd)
}

func validate(events []interpreter.Event) {
	cycles := parseIntoCycles(events)
	var allErrors []EventErrs
	for _, cycle := range cycles {
		_, cycleErrs := cycleValidators.Valid(cycle.Events)
		for _, event := range cycle.Events {
			_, err := eventValidator.Valid(event)
			allErrors = append(allErrors, EventErrs{
				event:     event,
				cycleID:   cycle.ID,
				cycleErrs: cycleErrs,
				eventErrs: err,
			})
		}
	}

	printValidationTable(allErrors)
}

func printValidationTable(info []EventErrs) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader([]string{"Cycle", "Cycle Errors", "Event Errors", "Boot-time", "Destination", "ID"})
	var data [][]string
	for _, eventErr := range info {
		data = append(data, getValidationRowInfo(eventErr))
	}
	table.SetAutoMergeCellsByColumnIndex([]int{0, 1, 3})
	table.SetRowLine(true)
	table.AppendBulk(data)
	table.Render()
}

func getValidationRowInfo(info EventErrs) []string {
	return []string{
		info.cycleID,
		errorTagsToString(info.cycleErrs),
		errorTagsToString(info.eventErrs),
		getBoottimeString(info.event),
		info.event.Destination,
		info.event.TransactionUUID,
	}
}

func createCycleValidators() history.CycleValidator {
	var metadataValidators []MetadataKey
	viper.UnmarshalKey("metadataValidators", &metadataValidators)

	validators := []history.CycleValidator{
		history.TransactionUUIDValidator(),
		history.SessionOnlineValidator(func(_ []interpreter.Event, _ string) bool { return false }),
		history.SessionOfflineValidator(func(_ []interpreter.Event, _ string) bool { return false }),
	}
	var withinCycleChecks []string
	var wholeCycleChecks []string
	for _, metadata := range metadataValidators {
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
	eventTypeValidator := validation.EventTypeValidator([]string{"reboot-pending", "offline", "online", "operational", "fully-manageable"})

	validators := validation.Validators([]validation.Validator{
		bootTimeValidator, birthdateValidator, birthdateAlignmentValidator, consistentIDValidator, bootDurationValidator, eventTypeValidator,
	})

	return validators
}

func createComparator() history.Comparator {
	return history.DefaultComparator()
}
