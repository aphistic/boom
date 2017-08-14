package boom

import (
	"github.com/efritz/glock"
)

type TaskConfig func(*taskConfig)

type taskConfig struct {
	clock glock.Clock
}

func newTaskConfig() *taskConfig {
	return &taskConfig{
		clock: glock.NewRealClock(),
	}
}

func (tc *taskConfig) ApplyConfigs(configs []TaskConfig) {
	for _, f := range configs {
		f(tc)
	}
}

func WithClock(clock glock.Clock) TaskConfig {
	return func(cfg *taskConfig) {
		cfg.clock = clock
	}
}
