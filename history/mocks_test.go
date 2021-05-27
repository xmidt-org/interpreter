package history

import (
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/interpreter"
	"github.com/xmidt-org/interpreter/validation"
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

type testTaggedError struct {
	tag  validation.Tag
	tags []validation.Tag
}

func (e testTaggedError) Error() string {
	return "test error"
}

func (e testTaggedError) Tag() validation.Tag {
	return e.tag
}

func (e testTaggedError) Tags() []validation.Tag {
	return e.tags
}
