package main

import (
	"time"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/history"
	"github.com/xmidt-org/interpreter/validation"
)

var (
	eventsValidator  validation.Validator
	cyclesValidators []history.CycleValidatorFunc
	comparator       history.Comparator
)

var ValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "validate a list of cycles and events and print",
	PreRun: func(cmd *cobra.Command, args []string) {
		eventsValidator = createEventValidators()
		cyclesValidators = createCycleValidators()
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
	event interpreter.Event
	cycleID string
	cycleErrs error
	eventErrs error
}

func init() {
	ParseCmd.AddCommand(ValidateCmd)
	RootCmd.AddCommand(ValidateCmd)
}

func validate(events []interpreter.Event) {
	cycles := parseIntoCycles(events, comparator)
	var allErrors []EventErrs
	for _, cycle := range cycles {
		
		for i, event := range cycle.Events {
			valid, err := eventsValidator.Valid(event)
			eventErrors[i] = validation.EventWithError{
				Event:       event,
				OriginalErr: err,
			}
		}
	}
}

func validateCycle(cycle BootCycle) error {
	var allErrors validation.Validators
	for _, validator := range cyclesValidators {
		if valid, 
	}
}

func createCycleValidators() []history.CycleValidatorFunc {
	var metadataValidators []MetadataKey
	viper.UnmarshalKey("metadataValidators", &metadataValidators)

	validators := []history.CycleValidatorFunc{
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
	eventTypeValidator := validation.EventTypeValidator([]string{"reboot-pending", "offline", "online", "operational", "fully-manageable"})

	validators := validation.Validators([]validation.Validator{
		bootTimeValidator, birthdateValidator, birthdateAlignmentValidator, consistentIDValidator, bootDurationValidator, eventTypeValidator,
	})

	return validators
}

func createComparator() history.Comparator {
	return history.DefaultComparator()
}
