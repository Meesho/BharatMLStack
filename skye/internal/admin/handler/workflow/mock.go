package workflow

import "github.com/stretchr/testify/mock"

type MockStateMachine struct {
	mock.Mock
}

func (m *MockStateMachine) ProcessStates(payload *ModelStateExecutorPayload) error {
	args := m.Called(payload)
	return args.Error(0)
}
