package workflow

type StateMachine interface {
	ProcessStates(payload *ModelStateExecutorPayload) error
}
