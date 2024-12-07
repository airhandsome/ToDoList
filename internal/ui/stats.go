package ui

import (
	"TodoList/internal/storage"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/theme"
)

type StatsView struct {
	container     *fyne.Container
	db            *storage.Database
	dateRange     *widget.Select
	taskStats     *widget.Label
	pomodoroStats *widget.Label
	refreshBtn    *widget.Button
}

func NewStatsView(db *storage.Database) *StatsView {
	sv := &StatsView{
		db:            db,
		taskStats:     widget.NewLabel(""),
		pomodoroStats: widget.NewLabel(""),
	}
	sv.setup()
	return sv
}

func (sv *StatsView) setup() {
	// 创建标题
	title := widget.NewLabelWithStyle("Statistics", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// 创建刷新按钮
	sv.refreshBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if selected := sv.dateRange.Selected; selected != "" {
			sv.updateStats(selected)
		}
	})

	// 创建日期范围选择器
	sv.dateRange = widget.NewSelect(
		[]string{"Today", "This Week", "This Month", "All Time"},
		func(selected string) {
			sv.updateStats(selected)
		},
	)

	// 创建顶部工具栏
	toolbar := container.NewHBox(
		widget.NewLabel("Time Range:"),
		sv.dateRange,
		sv.refreshBtn,
	)

	// 创建统计信息容器
	statsContainer := container.NewHBox(
		container.NewVBox(
			widget.NewLabelWithStyle("Task Statistics", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			sv.taskStats,
		),
		container.NewVBox(
			widget.NewLabelWithStyle("Pomodoro Statistics", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			sv.pomodoroStats,
		),
	)

	// 组织整体布局
	sv.container = container.NewVBox(
		title,
		toolbar,
		statsContainer,
	)

	// 设置默认选中值并更新统计
	sv.dateRange.SetSelected("Today")
}

func (sv *StatsView) updateStats(timeRange string) {
	var startDate, endDate time.Time
	now := time.Now()
	endDate = now

	// 根据选择的时间范围设置开始时间
	switch timeRange {
	case "Today":
		startDate = now.Truncate(24 * time.Hour)
	case "This Week":
		startDate = now.AddDate(0, 0, -int(now.Weekday()))
		startDate = startDate.Truncate(24 * time.Hour)
	case "This Month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case "All Time":
		startDate = time.Time{} // 零值表示不限制开始时间
	}

	// 获取任务统计
	taskStats, err := sv.db.GetTaskStats(startDate, endDate)
	if err != nil {
		fmt.Println("Error getting task stats:", err)
		return
	}

	// 获取番茄钟统计
	pomodoroStats, err := sv.db.GetPomodoroStats(startDate, endDate)
	if err != nil {
		fmt.Println("Error getting pomodoro stats:", err)
		return
	}

	// 计算完成率
	var completionRate float64
	if taskStats.TotalTasks > 0 {
		completionRate = float64(taskStats.CompletedTasks) / float64(taskStats.TotalTasks) * 100
	}

	// 更新任务统计显示
	sv.taskStats.SetText(fmt.Sprintf(
		"Total Tasks: %d\n"+
			"Completed: %d\n"+
			"Completion Rate: %.1f%%\n"+
			"Todo: %d\n"+
			"Doing: %d\n"+
			"Done: %d\n"+
			"Cancelled: %d",
		taskStats.TotalTasks,
		taskStats.CompletedTasks,
		completionRate,
		taskStats.TodoTasks,
		taskStats.DoingTasks,
		taskStats.DoneTasks,
		taskStats.CancelledTasks,
	))

	// 更新番茄钟统计显示
	sv.pomodoroStats.SetText(fmt.Sprintf(
		"Total Sessions: %d\n"+
			"Total Focus Time: %.1f hours\n"+
			"Average Session: %.1f minutes\n"+
			"Today's Sessions: %d\n"+
			"Today's Focus Time: %.1f hours",
		pomodoroStats.TotalSessions,
		float64(pomodoroStats.TotalDuration)/3600,
		pomodoroStats.AverageDuration/60,
		pomodoroStats.TodaySessions,
		float64(pomodoroStats.TodayDuration)/3600,
	))
}

func (sv *StatsView) Container() *fyne.Container {
	return sv.container
}
