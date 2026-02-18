package types

type ShadowState string

const (
	ShadowStateFree     ShadowState = "FREE"
	ShadowStateProcured ShadowState = "PROCURED"
)

type Action string

const (
	ActionProcure Action = "PROCURE"
	ActionRelease Action = "RELEASE"

	ActionIncrease Action = "INCREASE"
	ActionDecrease Action = "DECREASE"
	ActionResetTo0 Action = "RESET_TO_0"
)
