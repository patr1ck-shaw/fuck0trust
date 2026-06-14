//go:build walkgui
// +build walkgui

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall" // 👈 引入系统底层调用
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var (
	mainWindow   *walk.MainWindow
	statusLabel  *walk.Label
	noteEdit     *walk.LineEdit
	deviceIDText string
	ni           *walk.NotifyIcon
	
	// 👈 声明 Windows 原生 user32.dll 句柄
	user32           = syscall.NewLazyDLL("user32.dll")
	procIsIconic     = user32.NewProc("IsIconic")
	procShowWindow   = user32.NewProc("ShowWindow")
	procSetForeground = user32.NewProc("SetForegroundWindow")
)

func launchGUI() {
	// 启动日志
	logFile := filepath.Join(os.TempDir(), "fuck0trust_startup.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err == nil {
		fmt.Fprintf(f, "\n=== Startup at %s ===\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(f, "Step 1: Getting device ID...\n")
		f.Close()
	}
	
	if err := testMinimalWindow(); err != nil {
		walk.MsgBox(nil, "错误", "窗口系统初始化失败: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	
	deviceIDText = deviceID()
	shortDeviceID := deviceIDText
	if len(deviceIDText) > 32 {
		shortDeviceID = deviceIDText[:16] + "..." + deviceIDText[len(deviceIDText)-8:]
	}
	
	if err := (MainWindow{
		AssignTo:   &mainWindow,
		Title:      "Fuck0Trust",
		MinSize:    Size{Width: 560, Height: 420},
		MaxSize:    Size{Width: 560, Height: 420},
		Layout:     VBox{MarginsZero: true, SpacingZero: true},
		Background: SolidColorBrush{Color: walk.RGB(246, 247, 251)},
		
		// 👈 【Win32 修复】使用 IsIconic 检测是否点击了最小化按钮
		OnSizeChanged: func() {
			if mainWindow != nil {
				ret, _, _ := procIsIconic.Call(mainWindow.Handle())
				if ret != 0 {
					mainWindow.SetVisible(false) // 隐藏主窗口，从任务栏彻底消失
				}
			}
		},
		
		Children: []Widget{
			// 顶部蓝色标题栏
			Composite{
				Background: SolidColorBrush{Color: walk.RGB(37, 99, 235)},
				MinSize:    Size{Height: 82},
				MaxSize:    Size{Height: 82},
				Layout:     VBox{},
				Children: []Widget{
					Label{
						Text:       "Fuck0Trust",
						Font:       Font{Family: "Microsoft YaHei", PointSize: 24, Bold: true},
						TextColor:  walk.RGB(255, 255, 255),
						Background: SolidColorBrush{Color: walk.RGB(37, 99, 235)},
					},
				},
			},
			// 主内容区域 - 白色卡片
			Composite{
				Background: SolidColorBrush{Color: walk.RGB(246, 247, 251)},
				Layout:     VBox{Margins: Margins{Left: 22, Top: 18, Right: 22, Bottom: 18}},
				Children: []Widget{
					Composite{
						Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
						Layout:     VBox{Margins: Margins{Left: 22, Top: 18, Right: 22, Bottom: 18}, Spacing: 8},
						Children: []Widget{
							Label{
								AssignTo:   &statusLabel,
								Text:       "当前设备审批状态：检测中",
								Font:       Font{Family: "Microsoft YaHei", PointSize: 11, Bold: true},
								TextColor:  walk.RGB(51, 65, 85),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
								MinSize:    Size{Height: 30},
							},
							Label{
								Text:       "设备 ID：" + shortDeviceID,
								Font:       Font{Family: "Microsoft YaHei", PointSize: 9},
								TextColor:  walk.RGB(100, 116, 139),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
								MinSize:    Size{Height: 20},
							},
							VSpacer{Size: 8},
							Label{
								Text:       "联系方式（必填）：",
								Font:       Font{Family: "Microsoft YaHei", PointSize: 9},
								TextColor:  walk.RGB(51, 65, 85),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
							},
							LineEdit{
								AssignTo: &noteEdit,
								MaxSize:  Size{Height: 26},
							},
							VSpacer{Size: 10},
							Composite{
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
								Layout:     Grid{Columns: 3, Spacing: 8},
								Children: []Widget{
									PushButton{
										Text:      "提交审批",
										MinSize:   Size{Width: 140, Height: 32},
										OnClicked: guiRequestApproval,
									},
									PushButton{
										Text:      "同步审批状态",
										MinSize:   Size{Width: 140, Height: 32},
										OnClicked: guiSyncStatus,
									},
									PushButton{
										Text:      "执行一次",
										MinSize:   Size{Width: 140, Height: 32},
										OnClicked: guiRunOnce,
									},
									PushButton{
										Text:      "安装计划任务",
										MinSize:   Size{Width: 140, Height: 32},
										OnClicked: guiInstallTask,
									},
									PushButton{
										Text:      "删除计划任务",
										MinSize:   Size{Width: 140, Height: 32},
										OnClicked: guiRemoveTask,
									},
									Composite{
										Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
									},
								},
							},
							VSpacer{Size: 8},
							Label{
								Text:       "• 提示：首次使用请先提交审批，待管理员通过后再执行功能",
								Font:       Font{Family: "Microsoft YaHei", PointSize: 8},
								TextColor:  walk.RGB(148, 163, 184),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
							},
							Label{
								Text:       "• 同一设备 24 小时内只允许提交一次审批申请",
								Font:       Font{Family: "Microsoft YaHei", PointSize: 8},
								TextColor:  walk.RGB(148, 163, 184),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
							},
						},
					},
				},
			},
		},
	}.Create()); err != nil {
		walk.MsgBox(nil, "错误", "创建窗口失败: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	
	// 初始化系统托盘图标
	var errNi error
	ni, errNi = walk.NewNotifyIcon(mainWindow)
	if errNi == nil {
		ni.SetIcon(walk.IconInformation())
		ni.SetToolTip("Fuck0Trust 守护中")
		
		// 👈 【Win32 修复】双击托盘图标恢复时，直接向句柄发送 SW_RESTORE (9) 指令弹回主窗口
		ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
			if button == walk.LeftButton {
				mainWindow.SetVisible(true)
				procShowWindow.Call(mainWindow.Handle(), 9) // SW_RESTORE = 9
				procSetForeground.Call(mainWindow.Handle())
			}
		})
		ni.SetVisible(true)
	}

	defer func() {
		if ni != nil {
			ni.Dispose()
		}
	}()

	go func() {
		time.Sleep(300 * time.Millisecond)
		initialCheck()
	}()

	mainWindow.Run()
}

// 初始状态检查
func initialCheck() {
	mainWindow.Synchronize(func() {
		setStatusText("当前设备审批状态：同步中", walk.RGB(51, 65, 85))
	})

	if err := checkAPIReachable(8 * time.Second); err != nil {
		mainWindow.Synchronize(func() {
			setStatusText("当前设备审批状态：网络异常", walk.RGB(146, 64, 14))
			walk.MsgBox(mainWindow, "网络环境存在问题", 
				"当前网络无法访问审批服务，请更换网络或检查代理后重新打开客户端。", 
				walk.MsgBoxIconWarning)
		})
		return
	}

	status, err := refreshApprovalFromAPI(10 * time.Second)
	if err != nil {
		mainWindow.Synchronize(func() {
			setStatusText("当前设备审批状态：同步失败", walk.RGB(146, 64, 14))
		})
		return
	}

	if status.Blacklisted {
		mainWindow.Synchronize(func() {
			walk.MsgBox(mainWindow, "提示", "你已被拉黑，请联系 @pppatr1ck_bot", walk.MsgBoxIconError)
			os.Exit(0)
		})
		return
	}

	mainWindow.Synchronize(func() {
		updateStatusLabel(status.Approved)
	})
}

func setStatusText(text string, color walk.Color) {
	statusLabel.SetText(text)
	statusLabel.SetTextColor(color)
}

func updateStatusLabel(approved bool) {
	if approved {
		setStatusText("当前设备审批状态：已通过", walk.RGB(22, 101, 52))
	} else {
		setStatusText("当前设备审批状态：未通过/待审批", walk.RGB(153, 27, 27))
	}
}

func guiRequestApproval() {
	note := strings.TrimSpace(noteEdit.Text())
	if note == "" {
		walk.MsgBox(mainWindow, "提示", "请填写你的可联系方式，否则申请不予通过", walk.MsgBoxIconWarning)
		return
	}
	go func() {
		err := requestApproval(note)
		mainWindow.Synchronize(func() {
			if err != nil {
				walk.MsgBox(mainWindow, "执行失败", sanitizeError(err), walk.MsgBoxIconError)
				return
			}
			walk.MsgBox(mainWindow, "已提交", 
				"已提交待管理员审批。\n同一设备 24 小时内只能提交一次审批。", 
				walk.MsgBoxIconInformation)
		})
	}()
}

func guiSyncStatus() {
	go func() {
		status, err := refreshApprovalFromAPI(10 * time.Second)
		mainWindow.Synchronize(func() {
			if err != nil {
				walk.MsgBox(mainWindow, "执行失败", sanitizeError(err), walk.MsgBoxIconError)
				return
			}
			if status.Blacklisted {
				walk.MsgBox(mainWindow, "提示", "你已被拉黑，请联系 @pppatr1ck_bot", walk.MsgBoxIconError)
				os.Exit(0)
				return
			}
			updateStatusLabel(status.Approved)

			if status.Approved {
				walk.MsgBox(mainWindow, "审批状态", "当前设备审批状态：已通过", walk.MsgBoxIconInformation)
			} else {
				walk.MsgBox(mainWindow, "审批状态", "当前设备审批状态：未通过/待审批", walk.MsgBoxIconInformation)
			}
		})
	}()
}

func guiRunOnce() {
	go func() {
		err := runOnce()
		mainWindow.Synchronize(func() {
			if err != nil {
				walk.MsgBox(mainWindow, "执行失败", err.Error(), walk.MsgBoxIconError)
				return
			}
			walk.MsgBox(mainWindow, "执行完成", "受控功能已执行。", walk.MsgBoxIconInformation)
		})
	}()
}

func guiInstallTask() {
	go func() {
		err := installTask()
		mainWindow.Synchronize(func() {
			if err != nil {
				walk.MsgBox(mainWindow, "执行失败", err.Error(), walk.MsgBoxIconError)
				return
			}
			walk.MsgBox(mainWindow, "安装完成", 
				fmt.Sprintf("计划任务已创建/更新：%s，已开启开机自动常驻后台网络状态高级监测。", TaskName), 
				walk.MsgBoxIconInformation)
		})
	}()
}

func guiRemoveTask() {
	go func() {
		err := removeTask()
		mainWindow.Synchronize(func() {
			if err != nil {
				walk.MsgBox(mainWindow, "执行失败", err.Error(), walk.MsgBoxIconError)
				return
			}
			walk.MsgBox(mainWindow, "删除完成", 
				fmt.Sprintf("计划任务已删除：%s", TaskName), 
				walk.MsgBoxIconInformation)
		})
	}()
}