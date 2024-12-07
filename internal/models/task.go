package models

import (
	"time"
)

type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusCompleted TaskStatus = "completed"
)

type Task struct {
	ID          int64
	Title       string
	Description string
	Status      TaskStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
	Priority    int
	Date        string
}
