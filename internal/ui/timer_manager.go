package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"time"
)

type TimerManager struct {
	container *fyne.Container
	timers    []*PomodoroTimer
	addButton *widget.Button
}

func NewTimerManager() *TimerManager {
	tm := &TimerManager{
		timers: make([]*PomodoroTimer, 0),
	}

	tm.addButton = widget.NewButton("添加番茄钟", tm.showAddDialog)

	// 使用网格布局来展示多个番茄钟
	tm.container = container.NewVBox(
		tm.addButton,
		container.NewGridWithColumns(2), // 2列网格布局用于显示番茄钟
	)

	return tm
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

	pomodorosEntry := widget.NewEntry()
	pomodorosEntry.SetText("4")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "名称", Widget: nameEntry},
			{Text: "工作时长(分钟)", Widget: workEntry},
			{Text: "休息时长(分钟)", Widget: breakEntry},
			{Text: "长休息时长(分钟)", Widget: longBreakEntry},
			{Text: "长休息间隔(番茄钟数)", Widget: pomodorosEntry},
		},
		OnSubmit: func() {
			timer := NewPomodoroTimer(
				time.Duration(mustParseInt(workEntry.Text))*time.Minute,
				time.Duration(mustParseInt(breakEntry.Text))*time.Minute,
				time.Duration(mustParseInt(longBreakEntry.Text))*time.Minute,
			)

			// 设置删除回调
			timer.SetOnDelete(func() {
				tm.removeTimer(timer)
			})

			tm.timers = append(tm.timers, timer)
			tm.updateLayout()
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
		grid.Add(timer.container) // 添加计时器的容器到网格中
	}

	// 更新容器
	tm.container.Objects[1] = grid
	tm.container.Refresh()
}
