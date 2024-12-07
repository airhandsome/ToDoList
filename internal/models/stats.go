package models

import "time"

type TaskStats struct {
	TotalTasks     int
	CompletedTasks int
	CompletionRate float64
}

type PomodoroStats struct {
	TotalSessions   int
	TotalDuration   int64 // 以秒为单位
	AverageDuration int64
}

type PomodoroRecord struct {
	ID        int64
	TaskID    int64
	StartTime time.Time
	EndTime   time.Time
	Duration  int64 // 以秒为单位
}
