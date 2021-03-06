package validation

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/interpreter"
)

func TestDefaultValidator(t *testing.T) {
	assert := assert.New(t)
	validator := DefaultValidator()
	valid, err := validator.Valid(interpreter.Event{})
	assert.True(valid)
	assert.Nil(err)
}

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
	assert.Contains(err.Error(), "invalid event")
	assert.Contains(err.Error(), "another invalid event")
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
	year := 2015
	timeValidation := TimeValidator{ValidFrom: -2 * time.Hour, ValidTo: time.Hour, Current: currTime, MinValidYear: year}
	validator := BootTimeValidator(timeValidation)
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
			description: "Old boot-time",
			event: interpreter.Event{
				Birthdate: now.Add(-1 * time.Hour).UnixNano(),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(-3 * time.Hour).Unix()),
				},
			},
			valid:       false,
			expectedErr: InvalidBootTimeErr{OriginalErr: ErrPastDate, ErrorTag: OldBootTime},
		},
		{
			description: "Past 2015 boot-time",
			event: interpreter.Event{
				Birthdate: now.Add(-1 * time.Hour).UnixNano(),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(time.Date(year, now.Add(time.Hour*-24).Month(), now.Add(time.Hour*-24).Day(), 0, 0, 0, 0, time.Local).Unix()),
				},
			},
			valid:       false,
			expectedErr: InvalidBootTimeErr{OriginalErr: ErrInvalidYear, ErrorTag: InvalidBootTime},
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
			expectedErr: InvalidBootTimeErr{OriginalErr: ErrFutureDate, ErrorTag: InvalidBootTime},
		},
		{
			description: "No boot-time",
			event: interpreter.Event{
				Birthdate: now.Add(-1 * time.Hour).UnixNano(),
				Metadata:  map[string]string{},
			},
			valid:       false,
			expectedErr: InvalidBootTimeErr{ErrorTag: MissingBootTime},
		},
		{
			description: "Invalid boot-time format",
			event: interpreter.Event{
				Birthdate: now.Add(-1 * time.Hour).UnixNano(),
				Metadata: map[string]string{
					interpreter.BootTimeKey: "not a time stamp",
				},
			},
			valid:       false,
			expectedErr: InvalidBootTimeErr{ErrorTag: InvalidBootTime},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			valid, err := validator.Valid(tc.event)
			assert.Equal(tc.valid, valid)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				var taggedError TaggedError
				var expectedTaggedError TaggedError
				assert.True(errors.As(err, &taggedError))
				assert.True(errors.As(tc.expectedErr, &expectedTaggedError))
				assert.Contains(err.Error(), tc.expectedErr.Error())
				assert.Equal(expectedTaggedError.Tag(), taggedError.Tag())
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
			expectedErr: InvalidBirthdateErr{OriginalErr: ErrPastDate},
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
			expectedErr: InvalidBirthdateErr{OriginalErr: ErrFutureDate},
		},
		{
			description: "No birthdate",
			event: interpreter.Event{
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Add(2 * time.Hour).Unix()),
				},
			},
			valid:       false,
			expectedErr: InvalidBirthdateErr{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			valid, err := validator.Valid(tc.event)
			assert.Equal(tc.valid, valid)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}
}

func TestBirthdateAlignmentValidator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	testDuration := 60 * time.Second
	tests := []struct {
		description        string
		event              interpreter.Event
		expectedValid      bool
		duration           time.Duration
		expectedTimestamps []int64
	}{
		{
			description: "valid",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/mac:112233445566/%d", now.Add(5*time.Second).Unix()),
				Birthdate:   now.UnixNano(),
			},
			duration:      testDuration,
			expectedValid: true,
		},
		{
			description: "invalid",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/mac:112233445566/%d", now.Add(5*time.Minute).Unix()),
				Birthdate:   now.UnixNano(),
			},
			expectedValid:      false,
			duration:           testDuration,
			expectedTimestamps: []int64{now.Add(5 * time.Minute).Unix()},
		},
		{
			description: "multiple timestamps valid",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/mac:112233445566/%d/%d/something/%d", now.Add(5*time.Second).Unix(), now.Add(2*time.Second).Unix(), now.Add(10*time.Second).Unix()),
				Birthdate:   now.UnixNano(),
			},
			duration:      testDuration,
			expectedValid: true,
		},
		{
			description: "valid with negative duration",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/mac:112233445566/%d/%d/something/%d", now.Add(5*time.Second).Unix(), now.Add(2*time.Second).Unix(), now.Add(10*time.Second).Unix()),
				Birthdate:   now.UnixNano(),
			},
			duration:      testDuration * -1,
			expectedValid: true,
		},
		{
			description: "multiple timestamps invalid",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/mac:112233445566/%d/%d/something/%d", now.Add(5*time.Minute).Unix(), now.Add(2*time.Minute).Unix(), now.Add(10*time.Second).Unix()),
				Birthdate:   now.UnixNano(),
			},
			duration:           testDuration,
			expectedValid:      false,
			expectedTimestamps: []int64{now.Add(5 * time.Minute).Unix(), now.Add(2 * time.Minute).Unix()},
		},
		{
			description: "invalid with negative duration",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/mac:112233445566/%d/%d/something/%d", now.Add(5*time.Minute).Unix(), now.Add(2*time.Minute).Unix(), now.Add(10*time.Second).Unix()),
				Birthdate:   now.UnixNano(),
			},
			duration:           testDuration,
			expectedValid:      false,
			expectedTimestamps: []int64{now.Add(5 * time.Minute).Unix(), now.Add(2 * time.Minute).Unix()},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			val := BirthdateAlignmentValidator(tc.duration)
			valid, err := val.Valid(tc.event)
			assert.Equal(tc.expectedValid, valid)
			if tc.expectedValid {
				assert.Nil(err)
			} else {
				var birthdateErr InvalidBirthdateErr
				assert.True(errors.As(err, &birthdateErr))
				assert.ElementsMatch(tc.expectedTimestamps, birthdateErr.Timestamps)
				assert.Equal(tc.event.Destination, birthdateErr.Destination)
				assert.Equal(MisalignedBirthdate, birthdateErr.Tag())
			}
		})
	}
}

func TestDestinationValidator(t *testing.T) {
	tests := []struct {
		description       string
		event             interpreter.Event
		searchedEventType string
		valid             bool
		expectedErr       error
	}{
		{
			description: "Valid event",
			event: interpreter.Event{
				Destination: "event:device-status/mac:112233445566/some-event/random-string",
			},
			searchedEventType: "some-event",
			valid:             true,
		},
		{
			description: "Valid despite case",
			event: interpreter.Event{
				Destination: "event:device-status/mac:112233445566/SOME-EVENT/random-string",
			},
			searchedEventType: "some-event",
			valid:             true,
		},
		{
			description: "Valid despite case",
			event: interpreter.Event{
				Destination: "event:device-status/mac:112233445566/some-event/random-string",
			},
			searchedEventType: "SOME-EVENT",
			valid:             true,
		},
		{
			description: "event type mismatch",
			event: interpreter.Event{
				Destination: "event:device-status/mac:112233445566/random-event/random-string",
			},
			valid:             false,
			searchedEventType: "some-event",
			expectedErr:       ErrEventTypeMismatch,
		},
		{
			description: "Invalid event",
			event: interpreter.Event{
				Destination: "/random-event/",
			},
			searchedEventType: "some-event",
			expectedErr:       ErrNonEvent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			validator := DestinationValidator(tc.searchedEventType)
			valid, err := validator.Valid(tc.event)
			assert.Equal(tc.valid, valid)
			if tc.expectedErr == nil || err == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}

}

func TestConsistentDeviceIDValidator(t *testing.T) {
	val := ConsistentDeviceIDValidator()
	tests := []struct {
		description        string
		event              interpreter.Event
		expectedConsistent bool
		expectedIDs        []string
	}{
		{
			description: "pass",
			event: interpreter.Event{
				Source:      "mac:112233445566",
				Destination: "event:device-status/mac:112233445566/something-something",
				Metadata: map[string]string{
					"key": "some-value/mac:112233445566",
				},
			},
			expectedConsistent: true,
		},
		{
			description: "inconsistent source",
			event: interpreter.Event{
				Source:      "mac:112233445566/serial:12345678",
				Destination: "event:device-status/mac:112233445566/something-something",
				Metadata: map[string]string{
					"key": "some-value/mac:112233445566",
				},
			},
			expectedIDs:        []string{"mac:112233445566", "serial:12345678"},
			expectedConsistent: false,
		},
		{
			description: "inconsistent destination",
			event: interpreter.Event{
				Source:      "mac:112233445566",
				Destination: "event:device-status/mac:123/something-something",
				Metadata: map[string]string{
					"key": "some-value/mac:112233445566",
				},
			},
			expectedIDs:        []string{"mac:112233445566", "mac:123"},
			expectedConsistent: false,
		},
		{
			description: "inconsistent metadata",
			event: interpreter.Event{
				Source:      "mac:112233445566",
				Destination: "event:device-status/mac:112233445566/something-something",
				Metadata: map[string]string{
					"key": "some-value/mac:112233445566/serial:112233445566",
				},
			},
			expectedIDs:        []string{"mac:112233445566", "serial:112233445566"},
			expectedConsistent: false,
		},
		{
			description: "no source",
			event: interpreter.Event{
				Destination: "event:device-status/mac:112233445566/something-something",
				Metadata: map[string]string{
					"key": "some-value/mac:112233445566",
				},
			},
			expectedConsistent: true,
		},
		{
			description: "no destination",
			event: interpreter.Event{
				Source: "mac:112233445566",
				Metadata: map[string]string{
					"key": "some-value/mac:112233445566",
				},
			},
			expectedConsistent: true,
		},
		{
			description: "no id in metadata",
			event: interpreter.Event{
				Source:      "mac:112233445566",
				Destination: "event:device-status/mac:112233445566/something-something",
				Metadata: map[string]string{
					"key": "some-value",
				},
			},
			expectedConsistent: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			pass, err := val.Valid(tc.event)
			assert.Equal(tc.expectedConsistent, pass)
			if !tc.expectedConsistent {
				var e InconsistentIDErr
				assert.True(errors.As(err, &e))
				assert.Equal(InconsistentDeviceID, e.Tag())
				assert.ElementsMatch(tc.expectedIDs, e.IDs)
			} else {
				assert.Nil(err)
			}
		})
	}
}

func TestBootDurationValidator(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	tests := []struct {
		description string
		event       interpreter.Event
		duration    time.Duration
		valid       bool
		expectedErr error
		expectedTag Tag
	}{
		{
			description: "valid with timestamp",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/serial:112233445566/%d", now.Add(2*time.Minute).Unix()),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
			},
			duration: 10 * time.Second,
			valid:    true,
		},
		{
			description: "valid with multiple timestamps",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/serial:112233445566/%d/something-something/%d/%d", now.Add(2*time.Minute).Unix(), now.Add(3*time.Minute).Unix(), now.Add(time.Minute).Unix()),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
			},
			duration: 10 * time.Second,
			valid:    true,
		},
		{
			description: "valid with negative duration",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/serial:112233445566/%d/something-something/%d/%d", now.Add(2*time.Minute).Unix(), now.Add(3*time.Minute).Unix(), now.Add(time.Minute).Unix()),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
			},
			duration: -10 * time.Second,
			valid:    true,
		},
		{
			description: "valid with no timestamps",
			event: interpreter.Event{
				Destination: "event:device-status/serial:112233445566/",
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
			},
			duration: 10 * time.Second,
			valid:    true,
		},
		{
			description: "valid with no boot-time",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/serial:112233445566/%d", now.Add(5*time.Second).Unix()),
			},
			duration:    10 * time.Second,
			valid:       true,
			expectedErr: InvalidBootTimeErr{ErrorTag: MissingBootTime},
			expectedTag: MissingBootTime,
		},
		{
			description: "invalid",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/serial:112233445566/%d", now.Add(5*time.Second).Unix()),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
			},
			duration:    10 * time.Second,
			valid:       false,
			expectedErr: BootDurationErr{OriginalErr: ErrFastBoot, ErrorTag: FastBoot},
			expectedTag: FastBoot,
		},
		{
			description: "invalid with negative duration",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/serial:112233445566/%d", now.Add(5*time.Second).Unix()),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
			},
			duration:    -10 * time.Second,
			valid:       false,
			expectedErr: BootDurationErr{OriginalErr: ErrFastBoot, ErrorTag: FastBoot},
			expectedTag: FastBoot,
		},
		{
			description: "past timestamp",
			event: interpreter.Event{
				Destination: fmt.Sprintf("event:device-status/serial:112233445566/%d", now.Add(-5*time.Second).Unix()),
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
			},
			duration: 10 * time.Second,
			valid:    true,
		},
		{
			description: "regular int",
			event: interpreter.Event{
				Destination: "event:device-status/serial:112233445566/123",
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
			},
			duration: 10 * time.Second,
			valid:    true,
		},
		{
			description: "duration",
			event: interpreter.Event{
				Destination: "event:device-status/serial:112233445566/2s",
				Metadata: map[string]string{
					interpreter.BootTimeKey: fmt.Sprint(now.Unix()),
				},
			},
			duration: 10 * time.Second,
			valid:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			val := BootDurationValidator(tc.duration)
			valid, err := val.Valid(tc.event)
			assert.Equal(tc.valid, valid)
			if tc.expectedErr != nil {
				var taggedError TaggedError
				assert.Contains(err.Error(), tc.expectedErr.Error())
				assert.True(errors.As(err, &taggedError))
				assert.Equal(tc.expectedTag, taggedError.Tag())
			}
		})
	}
}

func TestEventTypeValidator(t *testing.T) {
	val := EventTypeValidator([]string{"online", "online", "offline"})
	tests := []struct {
		description   string
		event         interpreter.Event
		expectedValid bool
		expectedMatch string
		expectedErr   error
	}{
		{
			description: "valid",
			event: interpreter.Event{
				Destination: "event:device-status/mac:112233445566/online",
			},
			expectedValid: true,
		},
		{
			description: "invalid",
			event: interpreter.Event{
				Destination: "event:device-status/mac:112233445566/random",
			},
			expectedValid: false,
			expectedMatch: "random",
			expectedErr:   ErrInvalidEventType,
		},
		{
			description: "no type",
			event: interpreter.Event{
				Destination: "event:device-status/mac:112233445566",
			},
			expectedValid: false,
			expectedErr:   interpreter.ErrTypeNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			valid, err := val.Valid(tc.event)
			assert.Equal(tc.expectedValid, valid)
			if tc.expectedErr != nil {
				var invalidTypeErr InvalidDestinationErr
				assert.True(errors.Is(err, tc.expectedErr))
				assert.True(errors.As(err, &invalidTypeErr))
				assert.Equal(tc.expectedMatch, invalidTypeErr.EventType)
			}
		})
	}
}

func TestDeviceIDComparison(t *testing.T) {
	tests := []struct {
		checkID         string
		foundID         string
		consistent      bool
		expectedFoundID string
		expectedIDs     []string
	}{
		{
			checkID:         "event:device-status/serial:112233445566/something-something/123",
			foundID:         "serial:112233445566",
			consistent:      true,
			expectedFoundID: "serial:112233445566",
			expectedIDs:     []string{"serial:112233445566"},
		},
		{
			checkID:         "event:device-status/mac:112233445566/something-something/123",
			foundID:         "mac:112233445566",
			consistent:      true,
			expectedFoundID: "mac:112233445566",
			expectedIDs:     []string{"mac:112233445566"},
		},
		{
			checkID:         "event:device-status/uuid:112233445566/something-something/123",
			foundID:         "uuid:112233445566",
			consistent:      true,
			expectedFoundID: "uuid:112233445566",
			expectedIDs:     []string{"uuid:112233445566"},
		},
		{
			checkID:         "event:device-status/dns:112233445566/something-something/123",
			foundID:         "dns:112233445566",
			consistent:      true,
			expectedFoundID: "dns:112233445566",
			expectedIDs:     []string{"dns:112233445566"},
		},
		{
			checkID:         "event:device-status/mac:112233445566/something-something/mac:112233445566/123",
			foundID:         "mac:112233445566",
			consistent:      true,
			expectedFoundID: "mac:112233445566",
			expectedIDs:     []string{"mac:112233445566", "mac:112233445566"},
		},
		{
			checkID:         "event:device-status/mac:112233445566/something-something/mac:123/123",
			foundID:         "mac:112233445566",
			consistent:      false,
			expectedFoundID: "mac:112233445566",
			expectedIDs:     []string{"mac:112233445566", "mac:123"},
		},
		{
			checkID:         "mac:112233445566",
			foundID:         "mac:112233445566",
			consistent:      true,
			expectedFoundID: "mac:112233445566",
			expectedIDs:     []string{"mac:112233445566"},
		},
		{
			checkID:         "/mac:112233445566/",
			foundID:         "mac:112233445566",
			consistent:      true,
			expectedFoundID: "mac:112233445566",
			expectedIDs:     []string{"mac:112233445566"},
		},
		{
			checkID:         "dns:112233445566",
			foundID:         "dns:112233445566",
			consistent:      true,
			expectedFoundID: "dns:112233445566",
			expectedIDs:     []string{"dns:112233445566"},
		},
		{
			checkID:         "serial:112233445566",
			foundID:         "serial:112233445566",
			consistent:      true,
			expectedFoundID: "serial:112233445566",
			expectedIDs:     []string{"serial:112233445566"},
		},
		{
			checkID:         "uuid:112233445566",
			foundID:         "uuid:112233445566",
			consistent:      true,
			expectedFoundID: "uuid:112233445566",
			expectedIDs:     []string{"uuid:112233445566"},
		},
		{
			checkID:         "mac:112233445566",
			foundID:         "",
			consistent:      true,
			expectedFoundID: "mac:112233445566",
			expectedIDs:     []string{"mac:112233445566"},
		},
		{
			checkID:         "uuid:112233445566",
			foundID:         "mac:112233445566",
			consistent:      false,
			expectedFoundID: "mac:112233445566",
			expectedIDs:     []string{"uuid:112233445566"},
		},
		{
			checkID:         "mac:112233445566",
			foundID:         "mac:123",
			consistent:      false,
			expectedFoundID: "mac:123",
			expectedIDs:     []string{"mac:112233445566"},
		},
		{
			checkID:         "not-an-id",
			foundID:         "mac:123",
			consistent:      true,
			expectedFoundID: "mac:123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.checkID, func(t *testing.T) {
			assert := assert.New(t)
			ids := make(map[string]bool)
			consistent, id, ids := deviceIDComparison(tc.checkID, tc.foundID, ids)
			assert.Equal(tc.consistent, consistent)
			assert.Equal(tc.expectedFoundID, id)
			for _, id := range tc.expectedIDs {
				assert.True(ids[id])
			}
		})
	}
}

func TestGetBootTime(t *testing.T) {
	tests := []struct {
		description string
		event       interpreter.Event
		expectedTag Tag
	}{
		{
			description: "no boot-time",
			event:       interpreter.Event{},
			expectedTag: MissingBootTime,
		},
		{
			description: "unparsable boot-time",
			event: interpreter.Event{
				Metadata: map[string]string{
					interpreter.BootTimeKey: "not-an-int",
				},
			},
			expectedTag: InvalidBootTime,
		},
		{
			description: "negative boot-time",
			event: interpreter.Event{
				Metadata: map[string]string{
					interpreter.BootTimeKey: "-1",
				},
			},
			expectedTag: InvalidBootTime,
		},
		{
			description: "valid boot-time",
			event: interpreter.Event{
				Metadata: map[string]string{
					interpreter.BootTimeKey: "50",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			_, err := getBootTime(tc.event)
			if tc.expectedTag != Unknown {
				var taggedErr TaggedError
				assert.True(errors.As(err, &taggedErr))
				assert.Equal(tc.expectedTag, taggedErr.Tag())
			} else {
				assert.Nil(err)
			}
		})
	}
}
