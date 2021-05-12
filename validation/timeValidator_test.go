package validation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValid(t *testing.T) {
	now, err := time.Parse(time.RFC3339Nano, "2021-03-02T18:00:01Z")
	assert.Nil(t, err)
	currFunc := func() time.Time { return now }
	tests := []struct {
		description  string
		validFrom    time.Duration
		validTo      time.Duration
		minValidYear int
		maxValidYear int
		testTime     time.Time
		currTime     func() time.Time
		expectedRes  bool
		expectedErr  error
	}{
		{
			description: "Valid Time",
			validFrom:   -1 * time.Hour,
			validTo:     time.Hour,
			testTime:    now.Add(30 * time.Minute),
			currTime:    currFunc,
			expectedRes: true,
		},
		{
			description: "Nil Time Func",
			validFrom:   -1 * time.Hour,
			validTo:     time.Hour,
			testTime:    now.Add(30 * time.Minute),
			currTime:    nil,
			expectedRes: false,
			expectedErr: ErrNilTimeFunc,
		},
		{
			description: "Unix Time 0",
			validFrom:   -1 * time.Hour,
			validTo:     30 * time.Minute,
			testTime:    time.Unix(0, 0),
			currTime:    currFunc,
			expectedRes: false,
			expectedErr: ErrPastDate,
		},
		{
			description: "Before unix Time 0",
			validFrom:   -1 * time.Hour,
			validTo:     30 * time.Minute,
			testTime:    time.Unix(-10, 0),
			currTime:    currFunc,
			expectedRes: false,
			expectedErr: ErrPastDate,
		},
		{
			description: "Positive past buffer",
			validFrom:   time.Hour,
			validTo:     30 * time.Minute,
			currTime:    currFunc,
			testTime:    now.Add(2 * time.Minute),
			expectedRes: true,
		},
		{
			description: "0 buffers",
			validFrom:   0,
			validTo:     0,
			testTime:    now.Add(2 * time.Minute),
			currTime:    currFunc,
			expectedRes: false,
			expectedErr: ErrFutureDate,
		},
		{
			description: "Equal time",
			validFrom:   0,
			validTo:     0,
			testTime:    now,
			currTime:    currFunc,
			expectedRes: true,
		},
		{
			description: "Too far in past",
			validFrom:   -1 * time.Hour,
			validTo:     time.Hour,
			testTime:    now.Add(-2 * time.Hour),
			currTime:    currFunc,
			expectedRes: false,
			expectedErr: ErrPastDate,
		},
		{
			description: "Too far in future",
			validFrom:   -1 * time.Hour,
			validTo:     time.Hour,
			testTime:    now.Add(2 * time.Hour),
			currTime:    currFunc,
			expectedRes: false,
			expectedErr: ErrFutureDate,
		},
		{
			description:  "Past minValidYear",
			minValidYear: 2015,
			currTime:     currFunc,
			testTime:     time.Date(2014, now.Month(), now.Day(), 0, 0, 0, 0, time.Local),
			expectedRes:  false,
			expectedErr:  ErrInvalidYear,
		},
		{
			description:  "Past maxValidYear",
			maxValidYear: 2018,
			currTime:     currFunc,
			testTime:     now,
			expectedRes:  false,
			expectedErr:  ErrInvalidYear,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			tv := TimeValidator{Current: tc.currTime, ValidFrom: tc.validFrom, ValidTo: tc.validTo, MinValidYear: tc.minValidYear, MaxValidYear: tc.maxValidYear}
			valid, err := tv.Valid(tc.testTime)
			assert.Equal(tc.expectedRes, valid)
			if err == nil || tc.expectedErr == nil {
				assert.Equal(tc.expectedErr, err)
			} else {
				assert.Contains(err.Error(), tc.expectedErr.Error())
			}
		})
	}
}
