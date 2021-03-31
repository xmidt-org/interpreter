package history

import (
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/interpreter"
)

type mockComparator struct {
	mock.Mock
}

func (m *mockComparator) Compare(historyEvent interpreter.Event, incomingEvent interpreter.Event) (bool, error) {
	args := m.Called(historyEvent, incomingEvent)
	return args.Bool(0), args.Error(1)
}

type mockValidator struct {
	mock.Mock
}

func (m *mockValidator) Valid(e interpreter.Event) (bool, error) {
	args := m.Called(e)
	return args.Bool(0), args.Error(1)
}
