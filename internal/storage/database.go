package storage

import (
	"TodoList/internal/models"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func NewDatabase() (*Database, error) {
	db, err := sql.Open("sqlite3", "pomodoro.db")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	database := &Database{db: db}
	database.initTables()
	return database, nil
}

func (d *Database) initTables() error {
	// 创建任务表
	_, err := d.db.Exec(`
        CREATE TABLE IF NOT EXISTS tasks (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            title TEXT NOT NULL,
            description TEXT,
            status TEXT NOT NULL,
            created_at DATETIME NOT NULL,
            completed_at DATETIME,
            priority INTEGER NOT NULL,
            date TEXT NOT NULL DEFAULT CURRENT_DATE
        )
    `)
	if err != nil {
		return err
	}

	// 检查 date 列是否存在，如果不存在则添加
	var hasDateColumn bool
	err = d.db.QueryRow(`
        SELECT COUNT(*) > 0 
        FROM pragma_table_info('tasks') 
        WHERE name = 'date'
    `).Scan(&hasDateColumn)

	if err != nil {
		return err
	}

	if !hasDateColumn {
		_, err = d.db.Exec(`
            ALTER TABLE tasks 
            ADD COLUMN date TEXT NOT NULL DEFAULT CURRENT_DATE
        `)
		if err != nil {
			return err
		}
	}

	// 创建番茄钟记录表
	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS pomodoro_records (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            task_id INTEGER,
            start_time DATETIME NOT NULL,
            end_time DATETIME NOT NULL,
            duration INTEGER NOT NULL,
            FOREIGN KEY(task_id) REFERENCES tasks(id)
        )
    `)
	if err != nil {
		return err
	}

	// 创建番茄钟配置表
	_, err = d.db.Exec(`
        CREATE TABLE IF NOT EXISTS timer_configs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            work_duration INTEGER NOT NULL,
            break_duration INTEGER NOT NULL,
            long_break INTEGER NOT NULL,
            date TEXT NOT NULL
        )
    `)
	if err != nil {
		fmt.Printf("Error creating timer_configs table: %v\n", err)
		return err
	}
	return nil
}

// 任务相关方法
func (d *Database) SaveTask(task *models.Task) error {
	if task.ID == 0 {
		return d.insertTask(task)
	}
	return d.updateTask(task)
}

func (d *Database) insertTask(task *models.Task) error {
	result, err := d.db.Exec(`
        INSERT INTO tasks (title, description, status, created_at, completed_at, priority, date)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, task.Title, task.Description, task.Status, task.CreatedAt, task.CompletedAt, task.Priority, task.Date)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	task.ID = id
	return nil
}

func (d *Database) updateTask(task *models.Task) error {
	_, err := d.db.Exec(`
        UPDATE tasks 
        SET title = ?, description = ?, status = ?, completed_at = ?, priority = ?, date = ?
        WHERE id = ?
    `, task.Title, task.Description, task.Status, task.CompletedAt, task.Priority, task.Date, task.ID)
	return err
}

// 番茄钟记录相关方法
func (d *Database) SavePomodoroRecord(record *models.PomodoroRecord) error {
	_, err := d.db.Exec(`
        INSERT INTO pomodoro_records (task_id, start_time, end_time, duration)
        VALUES (?, ?, ?, ?)
    `, record.TaskID, record.StartTime, record.EndTime, record.Duration)
	return err
}

// 统计相关方法
type TaskStats struct {
	TotalTasks     int
	CompletedTasks int
	TodoTasks      int
	DoingTasks     int
	DoneTasks      int
	CancelledTasks int
}

type PomodoroStats struct {
	TotalSessions   int
	TotalDuration   int // 总时长（秒）
	TodaySessions   int
	TodayDuration   int     // 今日时长（秒）
	AverageDuration float64 // 平均时长（秒）
}

func (d *Database) GetTaskStats(startDate, endDate time.Time) (*TaskStats, error) {
	stats := &TaskStats{}

	query := `
        SELECT 
            COALESCE(COUNT(*), 0) as total,
            COALESCE(SUM(CASE WHEN status = 'DONE' THEN 1 ELSE 0 END), 0) as completed,
            COALESCE(SUM(CASE WHEN status = 'TODO' THEN 1 ELSE 0 END), 0) as todo,
            COALESCE(SUM(CASE WHEN status = 'DOING' THEN 1 ELSE 0 END), 0) as doing,
            COALESCE(SUM(CASE WHEN status = 'DONE' THEN 1 ELSE 0 END), 0) as done,
            COALESCE(SUM(CASE WHEN status = 'UNDO' THEN 1 ELSE 0 END), 0) as cancelled
        FROM (
            SELECT status 
            FROM tasks 
            WHERE date BETWEEN date(?) AND date(?)
            AND status IN ('TODO', 'DOING', 'DONE', 'UNDO')
        ) t
    `

	err := d.db.QueryRow(query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).Scan(
		&stats.TotalTasks,
		&stats.CompletedTasks,
		&stats.TodoTasks,
		&stats.DoingTasks,
		&stats.DoneTasks,
		&stats.CancelledTasks,
	)

	// 如果没有数据，返回全零的统计结果
	if err == sql.ErrNoRows {
		return &TaskStats{}, nil
	}

	return stats, err
}

func (d *Database) GetPomodoroStats(startDate, endDate time.Time) (*PomodoroStats, error) {
	stats := &PomodoroStats{}
	today := time.Now().Truncate(24 * time.Hour)

	// 获取总体统计
	err := d.db.QueryRow(`
        SELECT 
            COUNT(*) as sessions,
            COALESCE(SUM(duration), 0) as total_duration
        FROM pomodoro_records
        WHERE start_time BETWEEN ? AND ?
    `, startDate, endDate).Scan(&stats.TotalSessions, &stats.TotalDuration)
	if err != nil {
		return nil, err
	}

	// 获取今日统计
	err = d.db.QueryRow(`
        SELECT 
            COUNT(*) as today_sessions,
            COALESCE(SUM(duration), 0) as today_duration
        FROM pomodoro_records
        WHERE start_time >= ?
    `, today).Scan(&stats.TodaySessions, &stats.TodayDuration)

	return stats, err
}

func (d *Database) GetDistinctDates() ([]string, error) {
	var dates []string
	rows, err := d.db.Query("SELECT DISTINCT date FROM tasks ORDER BY date DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var date string
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}
	return dates, nil
}

func (d *Database) GetTasksByDate(date string) ([]*models.Task, error) {
	var tasks []*models.Task
	rows, err := d.db.Query(`
        SELECT id, title, description, status, created_at, completed_at, priority, date 
        FROM tasks 
        WHERE date = ?
        ORDER BY priority DESC, created_at DESC
    `, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		task := &models.Task{}
		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.CreatedAt,
			&task.CompletedAt,
			&task.Priority,
			&task.Date,
		); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (d *Database) CreateTask(task *models.Task) error {
	_, err := d.db.Exec(
		"INSERT INTO tasks (title, status, created_at, date) VALUES (?, ?, ?, ?)",
		task.Title, task.Status, task.CreatedAt, task.Date,
	)
	return err
}

func (d *Database) UpdateTask(task *models.Task) error {
	_, err := d.db.Exec(
		"UPDATE tasks SET title = ?, status = ?, completed_at = ? WHERE id = ?",
		task.Title, task.Status, task.CompletedAt, task.ID,
	)
	return err
}

// 添加删除任务的方法
func (d *Database) DeleteTask(taskID int64) error {
	_, err := d.db.Exec("DELETE FROM tasks WHERE id = ?", taskID)
	return err
}

// 添加保存配置的方法
func (d *Database) SaveTimerConfig(config *models.TimerConfig) error {
	fmt.Printf("Saving timer config: %+v\n", config)

	// 修复 SQL 语句，确保关键字之间有空格
	result, err := d.db.Exec(`
        INSERT INTO timer_configs 
        (name, work_duration, break_duration, long_break, date)
        VALUES (?, ?, ?, ?, ?)
    `, config.Name, 
       int64(config.WorkDuration.Seconds()), 
       int64(config.BreakDuration.Seconds()), 
       int64(config.LongBreak.Seconds()), 
       config.Date.Format("2006-01-02"))

    if err != nil {
        fmt.Printf("Error saving timer config: %v\n", err)
        return err
    }

    id, err := result.LastInsertId()
    if err != nil {
        return err
    }
    config.ID = id
    return nil
}

// 添加获取指定日期配置的方法
func (d *Database) GetTimerConfigsByDate(date time.Time) ([]*models.TimerConfig, error) {
	rows, err := d.db.Query(`
        SELECT id, name, work_duration, break_duration, long_break
        FROM timer_configs
        WHERE date = ?
    `, date.Format("2006-01-02"))

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.TimerConfig
	for rows.Next() {
		var config models.TimerConfig
		var workSeconds, breakSeconds, longBreakSeconds int64

		err := rows.Scan(
			&config.ID,
			&config.Name,
			&workSeconds,
			&breakSeconds,
			&longBreakSeconds,
		)
		if err != nil {
			return nil, err
		}

		config.WorkDuration = time.Duration(workSeconds) * time.Second
		config.BreakDuration = time.Duration(breakSeconds) * time.Second
		config.LongBreak = time.Duration(longBreakSeconds) * time.Second
		config.Date = date

		configs = append(configs, &config)
	}

	return configs, nil
}

// 添加删除配置的方法
func (d *Database) DeleteTimerConfig(name string, date time.Time) error {
	_, err := d.db.Exec(`
        DELETE FROM timer_configs 
        WHERE name = ? AND date = ?
    `, name, date.Format("2006-01-02"))
	return err
}

// 添加更新配置的方法
func (d *Database) UpdateTimerConfig(config *models.TimerConfig) error {
	_, err := d.db.Exec(`
        UPDATE timer_configs 
        SET work_duration = ?, break_duration = ?, long_break = ?
        WHERE name = ? AND date = ?
    `, int64(config.WorkDuration.Seconds()),
       int64(config.BreakDuration.Seconds()),
       int64(config.LongBreak.Seconds()),
       config.Name,
       config.Date.Format("2006-01-02"))
	return err
}
