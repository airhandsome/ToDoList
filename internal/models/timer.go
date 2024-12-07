package models

import (
	"time"
)

type TimerState int

const (
	StateIdle TimerState = iota
	StateRunning
	StatePaused
)

type Timer struct {
	Duration  time.Duration
	Remaining time.Duration
	State     TimerState
	Task      *Task // 关联的任务
}
