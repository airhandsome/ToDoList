package ui

import (
	"TodoList/internal/models"
	"TodoList/internal/storage"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"strconv"
	"time"
)

type TimerManager struct {
	container *fyne.Container
	timers    []*PomodoroTimer
	addButton *widget.Button
	db        *storage.Database
}

func NewTimerManager(db *storage.Database) *TimerManager {
	tm := &TimerManager{
		timers: make([]*PomodoroTimer, 0),
		db:     db,
	}

	tm.addButton = widget.NewButton("添加番茄钟", tm.showAddDialog)

	// 使用网格布局来展示多个番茄钟
	tm.container = container.NewVBox(
		tm.addButton,
		container.NewGridWithColumns(2), // 2列网格布局用于显示番茄钟
	)

	// 加载今天的配置
	tm.loadTodayConfigs()

	return tm
}

func (tm *TimerManager) loadTodayConfigs() {
	configs, err := tm.db.GetTimerConfigsByDate(time.Now())
	if err != nil {
		// 处理错误
		return
	}

	for _, config := range configs {
		timer := NewPomodoroTimer(
			config.Name,
			config.WorkDuration,
			config.BreakDuration,
			config.LongBreak,
		)
		timer.SetOnDelete(func() {
			tm.removeTimer(timer)
			tm.db.DeleteTimerConfig(config.Name, config.Date)
		})
		timer.SetOnSave(func() {
			tm.db.UpdateTimerConfig(config)
		})
		tm.timers = append(tm.timers, timer)
	}
	tm.updateLayout()
}

func (tm *TimerManager) saveTimerConfig(name string, work, break_, longBreak time.Duration) error {
	config := &models.TimerConfig{
		Name:          name,
		WorkDuration:  work,
		BreakDuration: break_,
		LongBreak:     longBreak,
		Date:          time.Now(),
	}
	return tm.db.SaveTimerConfig(config)
}

func (tm *TimerManager) showAddDialog() {
	w := fyne.CurrentApp().NewWindow("添加番茄钟")

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("番茄钟名称")

	workEntry := widget.NewEntry()
	workEntry.SetText("25")

	breakEntry := widget.NewEntry()
	breakEntry.SetText("5")

	longBreakEntry := widget.NewEntry()
	longBreakEntry.SetText("15")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "名称", Widget: nameEntry},
			{Text: "工作时长(分钟)", Widget: workEntry},
			{Text: "休息时长(分钟)", Widget: breakEntry},
			{Text: "长休息时长(分钟)", Widget: longBreakEntry},
		},
		OnSubmit: func() {
			fmt.Println("Form submitted")

			// 添加输入验证
			if nameEntry.Text == "" {
				dialog.ShowError(fmt.Errorf("请输入番茄钟名称"), w)
				return
			}

			// 打印所有输入值
			fmt.Printf("Input values - Name: %s, Work: %s, Break: %s, LongBreak: %s\n",
				nameEntry.Text, workEntry.Text, breakEntry.Text, longBreakEntry.Text)

			// 检查 TimerManager 的数据库连接
			if tm.db == nil {
				fmt.Println("ERROR: Database connection is nil")
				dialog.ShowError(fmt.Errorf("数据库连接失败"), w)
				return
			}

			workDuration := time.Duration(mustParseInt(workEntry.Text)) * time.Minute
			breakDuration := time.Duration(mustParseInt(breakEntry.Text)) * time.Minute
			longBreakDuration := time.Duration(mustParseInt(longBreakEntry.Text)) * time.Minute

			fmt.Printf("Parsed durations - Work: %v, Break:%v, LongBreak: %v\n",
				workDuration, breakDuration, longBreakDuration)

			// 保存配置到数据库
			err := tm.saveTimerConfig(
				nameEntry.Text,
				workDuration,
				breakDuration,
				longBreakDuration,
			)
			if err != nil {
				fmt.Printf("Error saving timer config: %v\n", err)
				dialog.ShowError(fmt.Errorf("保存配置失败:%v", err), w)
				return
			}

			timer := NewPomodoroTimer(
				nameEntry.Text,
				workDuration,
				breakDuration,
				longBreakDuration,
			)

			timer.SetOnDelete(func() {
				tm.removeTimer(timer)
			})
			timer.SetOnSave(func() {
				tm.db.UpdateTimerConfig(&models.TimerConfig{
					Name:          nameEntry.Text,
					WorkDuration:  workDuration,
					BreakDuration: breakDuration,
					LongBreak:     longBreakDuration,
					Date:          time.Now(),
				})
			})
			tm.timers = append(tm.timers, timer)
			tm.updateLayout()

			// 刷新整个容器
			tm.container.Refresh()

			w.Close()
		},
	}

	w.SetContent(form)
	w.Resize(fyne.NewSize(300, 300))
	w.Show()
}

func (tm *TimerManager) removeTimer(timer *PomodoroTimer) {
	// 停止计时器
	timer.Stop()

	// 从切片中移除
	for i, t := range tm.timers {
		if t == timer {
			tm.timers = append(tm.timers[:i], tm.timers[i+1:]...)
			break
		}
	}

	tm.updateLayout()
}

func (tm *TimerManager) updateLayout() {
	// 清除现有的网格布局
	grid := container.NewGridWithColumns(2)

	// 重新添加所有计时器
	for _, timer := range tm.timers {
		grid.Add(timer.container)
	}

	// 更新容器
	tm.container.Objects[1] = grid

	// 刷新整个容器
	tm.container.Refresh()
}

// 添加安全的整数解析函数
func mustParseInt(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		fmt.Printf("Error parsing int: %v\n", err)
		return 0
	}
	return i
}
