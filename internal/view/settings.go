package view

import "context"

type settingsKey string

const (
	// BasicModeKey is the key for the basic mode setting in the request context.
	BasicModeKey settingsKey = "basicMode"
)

// IsBasicMode returns true if the "basic mode" flag is set in the request context.
func IsBasicMode(ctx context.Context) bool {
	basic, ok := ctx.Value(BasicModeKey).(bool)
	return ok && basic
}
