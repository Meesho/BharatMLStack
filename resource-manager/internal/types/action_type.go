package types

import "strings"

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

type PoolEnv string

const (
	PoolEnvInt  PoolEnv = "int"
	PoolEnvProd PoolEnv = "prod"
)

func NormalizePoolEnv(env string) PoolEnv {
	return PoolEnv(strings.ToLower(strings.TrimSpace(env)))
}

func IsSupportedPoolEnv(env string) bool {
	n := NormalizePoolEnv(env)
	return n == PoolEnvInt || n == PoolEnvProd
}
