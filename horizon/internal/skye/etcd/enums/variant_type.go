package enums

// Type represents the if the model is currently in experiment or scaledup.
type Type string

// Constants representing different variant states.
const (
	SCALE_UP   Type = "SCALE_UP"
	EXPERIMENT Type = "EXPERIMENT"
)
