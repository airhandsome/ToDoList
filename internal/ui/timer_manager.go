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
	container   *fyne.Container
	timers     []*PomodoroTimer
	addButton  *widget.Button
	db         *storage.Database
	dateSelect *widget.Select
	currentDate time.Time
}

func NewTimerManager(db *storage.Database) *TimerManager {
	tm := &TimerManager{
		timers:      make([]*PomodoroTimer, 0),
		db:          db,
		currentDate: time.Now(),
	}

	tm.addButton = widget.NewButton("添加番茄钟", tm.showAddDialog)

	dates, err := tm.getAvailableDates()
	if err != nil {
		dates = []string{time.Now().Format("2006-01-02")}
	}

	tm.dateSelect = widget.NewSelect(dates, tm.onDateSelected)
	tm.dateSelect.SetSelected(time.Now().Format("2006-01-02"))

	toolbar := container.NewHBox(
		widget.NewLabel("选择日期:"),
		tm.dateSelect,
		tm.addButton,
	)

	tm.container = container.NewVBox(
		toolbar,
		container.NewGridWithColumns(2),
	)

	tm.loadDateConfigs(tm.currentDate)

	return tm
}

func (tm *TimerManager) getAvailableDates() ([]string, error) {
	dates, err := tm.db.GetDistinctDates()
	if err != nil {
		return nil, err
	}

	today := time.Now().Format("2006-01-02")
	hasToday := false
	for _, date := range dates {
		if date == today {
			hasToday = true
			break
		}
	}
	if !hasToday {
		dates = append([]string{today}, dates...)
	}

	return dates, nil
}

func (tm *TimerManager) onDateSelected(dateStr string) {
	selectedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}

	tm.currentDate = selectedDate
	defer tm.loadDateConfigs(selectedDate)
}

func (tm *TimerManager) loadDateConfigs(date time.Time) {
	tm.timers = make([]*PomodoroTimer, 0)

	configs, err := tm.db.GetTimerConfigsByDate(date)
	if err != nil {
		dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}

	for _, config := range configs {
		timer := NewPomodoroTimer(
			config.Name,
			config.WorkDuration,
			config.BreakDuration,
			config.LongBreak,
			tm.db,
		)
		currentTimer := timer
		timer.SetOnDelete(func() {
			tm.removeTimer(currentTimer)
		})
		tm.timers = append(tm.timers, timer)
	}

	defer tm.updateLayout()
}

func (tm *TimerManager) saveTimerConfig(name string, work, break_, longBreak time.Duration) error {
	config := &models.TimerConfig{
		Name:          name,
		WorkDuration:  work,
		BreakDuration: break_,
		LongBreak:     longBreak,
		Date:          tm.currentDate,
	}
	return tm.db.SaveTimerConfig(config)
}

func (tm *TimerManager) showAddDialog() {
	fmt.Println("Opening add dialog")
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

			if nameEntry.Text == "" {
				dialog.ShowError(fmt.Errorf("请输入番茄钟名称"), w)
				return
			}

			fmt.Printf("Input values - Name: %s, Work: %s, Break: %s, LongBreak: %s\n",
				nameEntry.Text, workEntry.Text, breakEntry.Text, longBreakEntry.Text)

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

			config := &models.TimerConfig{
				Name:          nameEntry.Text,
				WorkDuration:  workDuration,
				BreakDuration: breakDuration,
				LongBreak:     longBreakDuration,
				Date:          tm.currentDate,
			}

			err := tm.db.SaveTimerConfig(config)
			if err != nil {
				dialog.ShowError(fmt.Errorf("保存配置失败: %v", err), w)
				return
			}

			fmt.Println("Creating new timer")
			timer := NewPomodoroTimer(
				nameEntry.Text,
				workDuration,
				breakDuration,
				longBreakDuration,
				tm.db,
			)

			timer.SetOnDelete(func() {
				tm.removeTimer(timer)
			})

			tm.timers = append(tm.timers, timer)
			tm.updateLayout()

			tm.container.Refresh()

			w.Close()

			dates, _ := tm.getAvailableDates()
			tm.dateSelect.Options = dates
			tm.dateSelect.Refresh()
		},
	}

	w.SetContent(form)
	w.Resize(fyne.NewSize(300, 300))
	w.Show()
}

func (tm *TimerManager) removeTimer(timer *PomodoroTimer) {
	err := tm.db.DeleteTimerConfig(timer.name, tm.currentDate)
	if err != nil {
		dialog.ShowError(fmt.Errorf("删除配置失败: %v", err), fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}

	for i, t := range tm.timers {
		if t == timer {
			tm.timers = append(tm.timers[:i], tm.timers[i+1:]...)
			break
		}
	}

	tm.updateLayout()

	dates, _ := tm.getAvailableDates()
	tm.dateSelect.Options = dates
	tm.dateSelect.Refresh()
}

func (tm *TimerManager) updateLayout() {
	if tm.container == nil {
		return
	}

	if len(tm.container.Objects) < 2 {
		return
	}

	grid := container.NewGridWithColumns(2)

	for _, timer := range tm.timers {
		if timer != nil && timer.container != nil {
			grid.Add(timer.container)
		}
	}

	tm.container.Objects[1] = grid

	tm.container.Refresh()
}

func mustParseInt(s string) int64 {
	fmt.Printf("Parsing string: %s\n", s)
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		fmt.Printf("Error parsing int: %v\n", err)
		return 0
	}
	return i
}
