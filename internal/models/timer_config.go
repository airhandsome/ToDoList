package models

import "time"

type TimerConfig struct {
	ID            int64
	Name          string
	WorkDuration  time.Duration
	BreakDuration time.Duration
	LongBreak     time.Duration
	Date          time.Time
} 