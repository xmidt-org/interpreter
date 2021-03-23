package validation

import (
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/interpreter"
)

func TestValidators(t *testing.T) {
	assert := assert.New(t)
	testEvent := interpreter.Event{}
	validators := Validators([]Validator{testValidator(true, nil), testValidator(true, nil)})
	valid, err := validators.Valid(testEvent)
	assert.True(valid)
	assert.Nil(err)

	validators = Validators([]Validator{
		testValidator(true, nil),
		testValidator(false, errors.New("invalid event")),
		testValidator(false, errors.New("another invalid event")),
	})
	valid, err = validators.Valid(testEvent)
	assert.False(valid)
	assert.Equal(errors.New("invalid event"), err)
}

func testValidator(returnBool bool, returnErr error) ValidatorFunc {
	return func(e interpreter.Event) (bool, error) {
		return returnBool, returnErr
	}
}

func TestBootTimeValidator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	currTime := func() time.Time { return now }
	validation := TimeValidator{ValidFrom: -2 * time.Hour, ValidTo: time.Hour, Current: currTime}
	validator := BootTimeValidator(validation)
	tests := []struct {
		description string
		event       interpreter.Event
		valid       bool
		expectedErr error
	}{
		{
			description: "Valid event",
			event: interpreter.Event{
				Birthdate: now.Add(-1 * time.Hour).UnixNano(),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix()),
				},
			},
			valid: true,
		},
		{
			description: "Past boot-time",
			event: interpreter.Event{
				Birthdate: now.Add(-1 * time.Hour).UnixNano(),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(-3 * time.Hour).Unix()),
				},
			},
			valid:       false,
			expectedErr: InvalidEventErr{OriginalErr: InvalidBootTimeErr{OriginalErr: ErrPastDate}},
		},
		{
			description: "Future boot-time",
			event: interpreter.Event{
				Birthdate: now.Add(-1 * time.Hour).UnixNano(),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(2 * time.Hour).Unix()),
				},
			},
			valid:       false,
			expectedErr: InvalidEventErr{OriginalErr: InvalidBootTimeErr{OriginalErr: ErrFutureDate}},
		},
		{
			description: "No boot-time",
			event: interpreter.Event{
				Birthdate: now.Add(-1 * time.Hour).UnixNano(),
				Metadata:  map[string]string{},
			},
			valid:       false,
			expectedErr: InvalidEventErr{OriginalErr: InvalidBootTimeErr{}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			valid, err := validator(tc.event)
			assert.Equal(tc.valid, valid)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func TestBirthdateValidator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	currTime := func() time.Time { return now }
	validation := TimeValidator{ValidFrom: -2 * time.Hour, ValidTo: time.Hour, Current: currTime}
	validator := BirthdateValidator(validation)
	tests := []struct {
		description string
		event       interpreter.Event
		valid       bool
		expectedErr error
	}{
		{
			description: "Valid event",
			event: interpreter.Event{
				Birthdate: now.Add(-1 * time.Hour).UnixNano(),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix()),
				},
			},
			valid: true,
		},
		{
			description: "Past birthdate",
			event: interpreter.Event{
				Birthdate: now.Add(-3 * time.Hour).UnixNano(),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(-1 * time.Hour).Unix()),
				},
			},
			valid:       false,
			expectedErr: InvalidEventErr{OriginalErr: InvalidBirthdateErr{OriginalErr: ErrPastDate}},
		},
		{
			description: "Future birthdate",
			event: interpreter.Event{
				Birthdate: now.Add(2 * time.Hour).UnixNano(),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(2 * time.Hour).Unix()),
				},
			},
			valid:       false,
			expectedErr: InvalidEventErr{OriginalErr: InvalidBirthdateErr{OriginalErr: ErrFutureDate}},
		},
		{
			description: "No birthdate",
			event: interpreter.Event{
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(2 * time.Hour).Unix()),
				},
			},
			valid:       false,
			expectedErr: InvalidEventErr{OriginalErr: InvalidBirthdateErr{}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			valid, err := validator(tc.event)
			assert.Equal(tc.valid, valid)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func TestDestinationValidator(t *testing.T) {
	validator := DestinationValidator(regexp.MustCompile(".*/some-event/.*"))
	tests := []struct {
		description string
		event       interpreter.Event
		valid       bool
		expectedErr error
	}{
		{
			description: "Valid event",
			event: interpreter.Event{
				Destination: "some-prefix/device-id/some-event/112233445566/random",
			},
			valid: true,
		},
		{
			description: "Invalid event",
			event: interpreter.Event{
				Destination: "/random-event/",
			},
			expectedErr: InvalidEventErr{OriginalErr: ErrInvalidEventType},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			valid, err := validator(tc.event)
			assert.Equal(tc.valid, valid)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}

}

func TestNewestBootTimeValidator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	latestEvent := interpreter.Event{
		Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
		TransactionUUID: "123",
	}

	validator := NewestBootTimeValidator(latestEvent)
	tests := []struct {
		description string
		event       interpreter.Event
		valid       bool
		validator   Validator
		expectedErr error
	}{
		{
			description: "valid event",
			event: interpreter.Event{
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(-30 * time.Minute).Unix()),
				},
				TransactionUUID: "abc",
			},
			validator: validator,
			valid:     true,
		},
		{
			description: "same event uuid",
			event: interpreter.Event{
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(30 * time.Minute).Unix()),
				},
				TransactionUUID: "123",
			},
			validator: validator,
			valid:     true,
		},
		{
			description: "boot-time not present",
			event: interpreter.Event{
				Metadata:        map[string]string{},
				TransactionUUID: "abc",
			},
			validator: validator,
			valid:     true,
		},
		{
			description: "newer boot-time",
			event: interpreter.Event{
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(30 * time.Minute).Unix())},
				TransactionUUID: "abc",
			},
			validator:   validator,
			valid:       false,
			expectedErr: errNewerBootTime,
		},
		{
			description: "latest boot-time invalid",
			event: interpreter.Event{
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Add(-30 * time.Minute).Unix())},
				TransactionUUID: "abc",
			},
			validator: NewestBootTimeValidator(interpreter.Event{
				TransactionUUID: "123",
			}),
			valid:       false,
			expectedErr: errNewerBootTime,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			valid, err := tc.validator.Valid(tc.event)
			assert.Equal(tc.valid, valid)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.True(errors.Is(err, tc.expectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedErr),
				)
			}
		})
	}
}

func TestUniqueEventValidator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	destRegex := regexp.MustCompile(".*/online")
	latestEvent := interpreter.Event{
		Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
		TransactionUUID: "test",
		Birthdate:       now.UnixNano(),
	}

	validator := UniqueEventValidator(latestEvent, destRegex)
	tests := []struct {
		description string
		event       interpreter.Event
		valid       bool
		validator   Validator
		expectedErr error
	}{
		{
			description: "valid event",
			event: interpreter.Event{
				Destination: "mac:112233445566/online",
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(-30 * time.Minute).Unix()),
				},
				TransactionUUID: "abc",
				Birthdate:       now.Add(-30 * time.Minute).UnixNano(),
			},
			validator: validator,
			valid:     true,
		},
		{
			description: "same event uuid",
			event: interpreter.Event{
				Destination: "mac:112233445566/online",
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
				TransactionUUID: "test",
				Birthdate:       now.UnixNano(),
			},
			validator: validator,
			valid:     true,
		},
		{
			description: "event type mismatch",
			event: interpreter.Event{
				Destination: "mac:112233445566/offline",
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
				TransactionUUID: "abc",
				Birthdate:       now.UnixNano(),
			},
			validator: validator,
			valid:     true,
		},
		{
			description: "boot-time missing",
			event: interpreter.Event{
				Destination:     "mac:112233445566/online",
				Metadata:        map[string]string{},
				TransactionUUID: "abc",
				Birthdate:       now.UnixNano(),
			},
			validator: validator,
			valid:     true,
		},
		{
			description: "duplicate found, older birthdate",
			event: interpreter.Event{
				Destination:     "mac:112233445566/online",
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				TransactionUUID: "abc",
				Birthdate:       now.Add(-1 * time.Minute).UnixNano(),
			},
			validator:   validator,
			valid:       false,
			expectedErr: errDuplicateEvent,
		},
		{
			description: "duplicate found, same birthdate",
			event: interpreter.Event{
				Destination:     "mac:112233445566/online",
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				TransactionUUID: "abc",
				Birthdate:       now.UnixNano(),
			},
			validator:   validator,
			valid:       false,
			expectedErr: errDuplicateEvent,
		},
		{
			description: "duplicate found, later birthdate",
			event: interpreter.Event{
				Destination:     "mac:112233445566/online",
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				TransactionUUID: "abc",
				Birthdate:       now.Add(time.Minute).UnixNano(),
			},
			validator: validator,
			valid:     true,
		},
		{
			description: "latest boot-time invalid",
			event: interpreter.Event{
				Metadata:        map[string]string{interpreter.BootTimeKey: fmt.Sprint(now.Unix())},
				TransactionUUID: "123",
				Birthdate:       now.UnixNano(),
			},
			validator: UniqueEventValidator(interpreter.Event{
				TransactionUUID: "test",
				Birthdate:       now.UnixNano(),
			}, destRegex),
			valid: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			valid, err := tc.validator.Valid(tc.event)
			assert.Equal(tc.valid, valid)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.True(errors.Is(err, tc.expectedErr),
					fmt.Errorf("error [%v] doesn't contain error [%v] in its err chain",
						err, tc.expectedErr),
				)
			}
		})
	}
}
