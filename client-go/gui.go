//go:build walkgui
// +build walkgui

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		
		// 👈 【原生优化 1】直接通过 Walk 原生的 SizeChanged 捕获最小化，杜绝 Win32 API 转换导致的类型冲突
		OnSizeChanged: func() {
			if mainWindow != nil && mainWindow.AsFormBase().SizeState() == walk.FormMin {
				mainWindow.SetVisible(false) // 隐藏主窗口（完全不占任务栏）
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
	
	// 👈 【原生优化 2】初始化系统托盘图标逻辑，直接采用 Walk 内嵌的原生标准机制
	var errNi error
	ni, errNi = walk.NewNotifyIcon(mainWindow)
	if errNi == nil {
		ni.SetIcon(walk.IconInformation())
		ni.SetToolTip("Fuck0Trust 守护中")
		
		// 👈 【原生优化 3】双击托盘图标：采用最纯正安全的 FormNormal 指令拉回前台
		ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
			if button == walk.LeftButton && mainWindow != nil {
				mainWindow.SetVisible(true)
				mainWindow.AsFormBase().SetSizeState(walk.FormNormal)
				mainWindow.BringToTop()
			}
		})
		ni.SetVisible(true)
	}

	// 👈 【原生优化 4】主窗口关闭时安全销毁托盘，不再让其提前挂载在 launch 作用域内引发死锁
	mainWindow.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if ni != nil {
			ni.Dispose()
		}
	})

	// 👈 【原生优化 5】将核心同步检测挂载在窗口安全 Starting 消息后运行，彻底封死 Synchronize 抢跑闪退
	mainWindow.Starting().Attach(func() {
		go func() {
			time.Sleep(100 * time.Millisecond)
			initialCheck()
		}()
	})

	mainWindow.Run()
}

// 初始状态检查以及各类动作保持你原有的完美闭环
func initialCheck() {
	if mainWindow == nil || statusLabel == nil {
		return
	}
	
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
	if statusLabel != nil {
		statusLabel.SetText(text)
		statusLabel.SetTextColor(color)
	}
}

func updateStatusLabel(approved bool) {
	if approved {
		setStatusText("当前设备审批状态：已通过", walk.RGB(22, 101, 52))
	} else {
		setStatusText("当前设备审批状态：未通过/待审批", walk.RGB(153, 27, 27))
	}
}

func guiRequestApproval() {
	if noteEdit == nil {
		return
	}
	note := strings.TrimSpace(noteEdit.Text())
	if note == "" {
		walk.MsgBox(mainWindow, "提示", "请填写你的可联系方式，否则申请不予通过", walk.MsgBoxIconWarning)
		return
	}
	go func() {
		err := requestApproval(note)
		if mainWindow == nil {
			return
		}
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
		if mainWindow == nil {
			return
		}
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
		if mainWindow == nil {
			return
		}
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
		if mainWindow == nil {
			return
		}
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
		if mainWindow == nil {
			return
		}
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