package ui

import (
	"fmt"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/faiface/beep/effects"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"image/color"
	"os"
	"sync"
)

// PomodoroTimer 表示一个番茄钟计时器
type PomodoroTimer struct {
	workDuration      time.Duration       // 工作时长
	breakDuration     time.Duration       // 休息时长
	longBreakDuration time.Duration       // 长休息时长
	isWorking         bool                // 是否处于工作状态
	isRunning         bool                // 是否正在运行
	remainingTime     time.Duration       // 剩余时间
	onTick            func(time.Duration) // 计时回调函数
	onComplete        func()              // 完成回调函数

	// UI 组件
	container      *fyne.Container
	timeLabel      *canvas.Text
	startButton    *widget.Button
	resetButton    *widget.Button
	statusLabel    *canvas.Text
	countLabel     *canvas.Text  // 显示完成的番茄钟数量
	settingsButton *widget.Button // 设置按钮

	// 新增字段
	pomodoroCount           int            // 完成的番茄钟数量
	pomodorosUntilLongBreak int            // 到长休息还需要的番茄钟数
	name                    string         // 添加名称字段
	SetDeleteCallback       func()         // 用于设置删除回调
	onDelete                func()         // 删除回调函数
	deleteBtn               *widget.Button // 删除按钮
}
type SoundEffect int

const (
	SoundWorkComplete SoundEffect = iota
	SoundBreakComplete
	SoundLongBreakComplete
)

// 添加全局变量用于音频初始化
var (
	audioOnce    sync.Once
	soundBuffers map[SoundEffect]*beep.Buffer
	volume       float64 = 1.0
)

// 初始化音频系统
func initAudio() error {
	soundBuffers = make(map[SoundEffect]*beep.Buffer)

	sounds := map[SoundEffect]string{
		SoundWorkComplete:      "assets/work_complete.wav",
		SoundBreakComplete:     "assets/break_complete.wav",
		SoundLongBreakComplete: "assets/long_break_complete.wav",
	}

	var format beep.Format
	for effect, path := range sounds {
		f, err := os.Open(path)
		if err != nil {
			return err
		}

		streamer, fmt, err := wav.Decode(f)
		if err != nil {
			f.Close()
			return err
		}

		if format.SampleRate == 0 {
			format = fmt
			if err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)); err != nil {
				return err
			}
		}

		buffer := beep.NewBuffer(fmt)
		buffer.Append(streamer)
		soundBuffers[effect] = buffer

		streamer.Close()
		f.Close()
	}

	return nil
}

// 播放指定的音效
func playSound(effect SoundEffect) {
	if buffer, ok := soundBuffers[effect]; ok {
		streamer := buffer.Streamer(0, buffer.Len())

		// 创建音量控制器
		volumeCtrl := &effects.Volume{
			Streamer: streamer,
			Base:     2,
			Volume:   volume,
			Silent:   false,
		}

		speaker.Play(volumeCtrl)
	}
}

// 设置音量
func setVolume(v float64) {
	volume = v
}

// 定义颜色常量
var (
	workColor            = color.NRGBA{R: 255, G: 99, B: 99, A: 180}   // 半透明红色
	breakColor           = color.NRGBA{R: 76, G: 175, B: 80, A: 180}   // 半透明绿色
	longBreakColor       = color.NRGBA{R: 33, G: 150, B: 243, A: 180}  // 半透明蓝色
	borderColor          = color.NRGBA{R: 200, G: 200, B: 200, A: 255} // 不透明边框
	buttonPrimaryColor   = color.NRGBA{R: 255, G: 64, B: 129, A: 255}  // 粉红色
	buttonSecondaryColor = color.NRGBA{R: 68, G: 138, B: 255, A: 255}  // 蓝色
	textColor            = color.NRGBA{R: 40, G: 40, B: 40, A: 255}      // 深灰色文本
	timeColor            = color.NRGBA{R: 25, G: 25, B: 25, A: 255}      // 更深的时间文本
	countColor           = color.NRGBA{R: 80, G: 80, B: 80, A: 255}     // 较浅的计数文本
)

// 定义背景图片路径
const (
	workBgPath      = "assets/backgrounds/work.jpg"
	breakBgPath     = "assets/backgrounds/break.jpg"
	longBreakBgPath = "assets/backgrounds/long_break.jpg"
)

// NewPomodoroTimer 创建一个新的番茄钟计时器
func NewPomodoroTimer(workDuration, breakDuration, longBreakDuration time.Duration) *PomodoroTimer {
	p := &PomodoroTimer{
		workDuration:            workDuration,
		breakDuration:           breakDuration,
		longBreakDuration:       longBreakDuration,
		isWorking:               true,
		remainingTime:           workDuration,
		pomodorosUntilLongBreak: 4,
	}

	// 创建背景图片
	background := canvas.NewImageFromFile(workBgPath)
	//background := canvas.NewRectangle(workColor)
	background.Resize(fyne.NewSize(300, 200)) // 设置合适的大小
	background.FillMode = canvas.ImageFillStretch

	// 创建半透明遮罩，使背景不那么显眼
	overlay := canvas.NewRectangle(color.NRGBA{R: 255, G: 255, B: 255, A: 180})
	overlay.CornerRadius = 20

	// 初始化 UI 组件
	p.timeLabel = canvas.NewText(formatDuration(workDuration), timeColor)
	p.timeLabel.TextStyle = fyne.TextStyle{Bold: true}
	p.timeLabel.TextSize = 32
	p.timeLabel.Alignment = fyne.TextAlignCenter

	p.statusLabel = canvas.NewText("工作时间", textColor)
	p.statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	p.statusLabel.TextSize = 20
	p.statusLabel.Alignment = fyne.TextAlignCenter

	p.countLabel = canvas.NewText("已完成: 0 个番茄钟", countColor)
	p.countLabel.TextSize = 16
	p.countLabel.Alignment = fyne.TextAlignCenter

	// 创建删除按钮
	p.deleteBtn = widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if p.onDelete != nil {
			p.onDelete()
		}
	})
	p.deleteBtn.Importance = widget.HighImportance

	// 创建主要控制按钮并设置样式
	p.startButton = widget.NewButtonWithIcon("开始", theme.MediaPlayIcon(), func() {
		if p.isRunning {
			p.Stop()
			p.startButton.SetIcon(theme.MediaPlayIcon())
			p.startButton.SetText("开始")
		} else {
			p.Start()
			p.startButton.SetIcon(theme.MediaPauseIcon())
			p.startButton.SetText("停止")
		}
	})
	p.startButton.Importance = widget.HighImportance

	p.resetButton = widget.NewButtonWithIcon("重置", theme.MediaReplayIcon(), p.Reset)
	p.resetButton.Importance = widget.MediumImportance

	p.settingsButton = widget.NewButtonWithIcon("设置", theme.SettingsIcon(), p.showSettings)
	p.settingsButton.Importance = widget.MediumImportance

	// 创建顶部栏（包含状态标签和删除按钮）
	topBar := container.NewBorder(
		nil, nil, nil, p.deleteBtn,
		container.NewPadded(p.statusLabel),
	)

	// 创建控制按钮容器
	controls := container.NewHBox(
		p.startButton,
		p.resetButton,
		p.settingsButton,
	)

	// 创建内容容器
	content := container.NewVBox(
		topBar,
		container.NewPadded(p.timeLabel),
		container.NewPadded(p.countLabel),
		controls,
	)

	// 添加内边距
	paddedContent := container.NewPadded(content)

	// 创建主容器，注意层次顺序
	p.container = container.NewMax(
		background, // 最底层：背景图片
		overlay,    // 中间层：半透明遮罩
		//border,        // 上层：边框
		paddedContent, // 最上层：内容
	)

	// 设置回调
	p.SetOnTick(func(d time.Duration) {
		p.timeLabel.Text = formatDuration(d)
		p.timeLabel.Refresh()
	})

	p.SetOnComplete(func() {
		if p.isWorking {
			p.statusLabel.Text = "休息时间"
			background := canvas.NewImageFromFile(breakBgPath)
			background.Resize(fyne.NewSize(300, 200)) // 设置合适的大小
			background.FillMode = canvas.ImageFillStretch
		} else {
			p.statusLabel.Text = "工作时间"
			background := canvas.NewImageFromFile(workBgPath)
			background.Resize(fyne.NewSize(300, 200)) // 设置合适的大小
			background.FillMode = canvas.ImageFillStretch
		}
		background.Refresh()
	})

	return p
}

func (p *PomodoroTimer) toggleTimer() {
	if p.isRunning {
		p.Stop()
		p.startButton.SetText("开始")
	} else {
		p.Start()
		p.startButton.SetText("停止")
	}
}

// formatDuration 将时间转换为显示格式
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// Start 开始计时
func (p *PomodoroTimer) Start() {
	if p.isRunning {
		return
	}
	p.isRunning = true

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for p.isRunning && p.remainingTime > 0 {
			<-ticker.C
			p.remainingTime -= time.Second
			if p.onTick != nil {
				p.onTick(p.remainingTime)
			}

			if p.remainingTime <= 0 {
				if p.onComplete != nil {
					p.onComplete()
				}
				p.Toggle()
			}
		}
	}()
}

// Stop 停止计时
func (p *PomodoroTimer) Stop() {
	p.isRunning = false
}

// Reset 重置计时器
func (p *PomodoroTimer) Reset() {
	p.Stop()
	if p.isWorking {
		p.remainingTime = p.workDuration
	} else {
		p.remainingTime = p.breakDuration
	}
}

// Toggle 切换工作/休息状态
func (p *PomodoroTimer) Toggle() {
	if p.isWorking {
		p.pomodoroCount++
		p.countLabel.Text = fmt.Sprintf("已完成: %d 个番茄钟", p.pomodoroCount)
		p.countLabel.Refresh()

		if p.pomodoroCount%p.pomodorosUntilLongBreak == 0 {
			p.remainingTime = p.longBreakDuration
			p.statusLabel.Text = "长休息时间"
			// 设置长休息背景
			if bg, ok := p.container.Objects[0].(*canvas.Image); ok {
				bg.File = longBreakBgPath
				bg.Refresh()
			}
		} else {
			p.remainingTime = p.breakDuration
			p.statusLabel.Text = "休息时间"
			// 设置休息背景
			if bg, ok := p.container.Objects[0].(*canvas.Image); ok {
				bg.File = breakBgPath
				bg.Refresh()
			}
		}
	} else {
		p.remainingTime = p.workDuration
		p.statusLabel.Text = "工作时间"
		// 设置工作背景
		if bg, ok := p.container.Objects[0].(*canvas.Image); ok {
			bg.File = workBgPath
			bg.Refresh()
		}
	}
	p.statusLabel.Refresh()
	p.isWorking = !p.isWorking

	// 播放提示音
	go p.playNotificationSound()
}

// SetOnTick 设置计时回调函数
func (p *PomodoroTimer) SetOnTick(callback func(time.Duration)) {
	p.onTick = func(d time.Duration) {
		if callback != nil {
			callback(d)
		}
		p.timeLabel.Text = formatDuration(d)
		p.timeLabel.Refresh()
	}
}

// SetOnComplete 设置完成回调函数
func (p *PomodoroTimer) SetOnComplete(callback func()) {
	p.onComplete = callback
}

// IsWorking 返回是否处于工作状态
func (p *PomodoroTimer) IsWorking() bool {
	return p.isWorking
}

// IsRunning 返回是否正在运行
func (p *PomodoroTimer) IsRunning() bool {
	return p.isRunning
}

// GetRemainingTime 获取剩余时间
func (p *PomodoroTimer) GetRemainingTime() time.Duration {
	return p.remainingTime
}

// showSettings 显示设置窗口
func (p *PomodoroTimer) showSettings() {
	// 创建设置窗口
	w := fyne.CurrentApp().NewWindow("番茄钟设置")

	workEntry := widget.NewEntry()
	workEntry.SetText(fmt.Sprintf("%d", int(p.workDuration.Minutes())))

	breakEntry := widget.NewEntry()
	breakEntry.SetText(fmt.Sprintf("%d", int(p.breakDuration.Minutes())))

	longBreakEntry := widget.NewEntry()
	longBreakEntry.SetText(fmt.Sprintf("%d", int(p.longBreakDuration.Minutes())))

	pomodorosEntry := widget.NewEntry()
	pomodorosEntry.SetText(fmt.Sprintf("%d", p.pomodorosUntilLongBreak))

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "工作时长(分钟)", Widget: workEntry},
			{Text: "休息时长(分钟)", Widget: breakEntry},
			{Text: "长休息时长(分钟)", Widget: longBreakEntry},
			{Text: "长休息间隔(番茄钟数)", Widget: pomodorosEntry},
		},
		OnSubmit: func() {
			// 保存设置
			p.workDuration = time.Duration(mustParseInt(workEntry.Text)) * time.Minute
			p.breakDuration = time.Duration(mustParseInt(breakEntry.Text)) * time.Minute
			p.longBreakDuration = time.Duration(mustParseInt(longBreakEntry.Text)) * time.Minute
			p.pomodorosUntilLongBreak = mustParseInt(pomodorosEntry.Text)
			p.Reset()
			w.Close()
		},
	}

	w.SetContent(form)
	w.Resize(fyne.NewSize(300, 200))
	w.Show()
}

// 辅助函数
func mustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func (p *PomodoroTimer) playNotificationSound() {
	// 根据当前状态播放不同的音效
	if p.isWorking {
		// 工作时间结束，播放工作完成音效
		playSound(SoundWorkComplete)
	} else {
		// 休息时间结束，播放休息完成音效
		playSound(SoundBreakComplete)
	}
}

// 添加设置删除回调的方法
func (p *PomodoroTimer) SetOnDelete(callback func()) {
	p.onDelete = callback
}
