package z

import "sync/atomic"

// ErrorMap customizes issue messages. Return "" to defer to the next map in
// the chain (`undefined` return).
type ErrorMap func(iss *Issue) string

// MessageFromString adapts a fixed message to an ErrorMap, mirroring's
// shorthand `z.string().min(5, "too short")`.
func MessageFromString(msg string) ErrorMap {
	if msg == "" {
		return nil
	}
	return func(*Issue) string { return msg }
}

// Config is the global configuration (z.config()).
type Config struct {
	// CustomError overrides locale messages globally (third link in chain).
	CustomError ErrorMap
	// LocaleError is the lowest-priority message source. Defaults to EnLocale.
	LocaleError ErrorMap
}

var globalConfig atomic.Pointer[Config]

func init() {
	globalConfig.Store(&Config{})
}

// Configure atomically replaces the global config and returns the previous
// one. Pass nil fields to clear them. Safe for concurrent use.
func Configure(cfg Config) Config {
	prev := globalConfig.Load()
	globalConfig.Store(&cfg)
	return *prev
}

// GetConfig returns the current global config.
func GetConfig() *Config {
	return globalConfig.Load()
}
