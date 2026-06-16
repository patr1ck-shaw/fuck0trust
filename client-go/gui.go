//go:build walkgui
// +build walkgui

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

var (
	mainWindow   *walk.MainWindow
	statusLabel  *walk.Label
	noteEdit     *walk.LineEdit
	deviceIDText string
	ni           *walk.NotifyIcon
)

func launchGUI() {
	deviceIDText = deviceID()
	shortDeviceID := deviceIDText
	if len(deviceIDText) > 32 {
		shortDeviceID = deviceIDText[:16] + "..." + deviceIDText[len(deviceIDText)-8:]
	}

	if err := (MainWindow{
		AssignTo:   &mainWindow,
		Title:      "Fuck0Trust 设备审批客户端",
		MinSize:    Size{Width: 600, Height: 480},
		MaxSize:    Size{Width: 600, Height: 480},
		Layout:     VBox{MarginsZero: true, SpacingZero: true},
		Background: SolidColorBrush{Color: walk.RGB(248, 250, 252)},

		Children: []Widget{
			// 顶部标题栏
			Composite{
				Background: SolidColorBrush{Color: walk.RGB(30, 58, 138)},
				MinSize:    Size{Height: 100},
				MaxSize:    Size{Height: 100},
				Layout:     VBox{Margins: Margins{Left: 30, Top: 20, Right: 30, Bottom: 20}},
				Children: []Widget{
					Label{
						Text:       "🛡️ Fuck0Trust",
						Font:       Font{Family: "Microsoft YaHei UI", PointSize: 20, Bold: true},
						TextColor:  walk.RGB(255, 255, 255),
						Background: SolidColorBrush{Color: walk.RGB(30, 58, 138)},
					},
					Label{
						Text:       "设备审批与网络守护系统",
						Font:       Font{Family: "Microsoft YaHei UI", PointSize: 9},
						TextColor:  walk.RGB(191, 219, 254),
						Background: SolidColorBrush{Color: walk.RGB(30, 58, 138)},
					},
				},
			},
			// 主内容区域
			Composite{
				Background: SolidColorBrush{Color: walk.RGB(248, 250, 252)},
				Layout:     VBox{Margins: Margins{Left: 30, Top: 20, Right: 30, Bottom: 20}, Spacing: 16},
				Children: []Widget{
					// 状态卡片
					Composite{
						Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
						Layout:     VBox{Margins: Margins{Left: 24, Top: 20, Right: 24, Bottom: 20}, Spacing: 12},
						Children: []Widget{
							Label{
								AssignTo:   &statusLabel,
								Text:       "设备审批状态：检测中...",
								Font:       Font{Family: "Microsoft YaHei UI", PointSize: 11, Bold: true},
								TextColor:  walk.RGB(71, 85, 105),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
								MinSize:    Size{Height: 28},
							},
							Composite{
								Background: SolidColorBrush{Color: walk.RGB(241, 245, 249)},
								Layout:     VBox{Margins: Margins{Left: 12, Top: 8, Right: 12, Bottom: 8}},
								MinSize:    Size{Height: 36},
								Children: []Widget{
									Label{
										Text:       "设备 ID: " + shortDeviceID,
										Font:       Font{Family: "Consolas", PointSize: 9},
										TextColor:  walk.RGB(100, 116, 139),
										Background: SolidColorBrush{Color: walk.RGB(241, 245, 249)},
									},
								},
							},
						},
					},
					// 审批申请区域
					Composite{
						Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
						Layout:     VBox{Margins: Margins{Left: 24, Top: 20, Right: 24, Bottom: 20}, Spacing: 10},
						Children: []Widget{
							Label{
								Text:       "联系方式（必填）",
								Font:       Font{Family: "Microsoft YaHei UI", PointSize: 9, Bold: true},
								TextColor:  walk.RGB(51, 65, 85),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
							},
							LineEdit{
								AssignTo: &noteEdit,
								MinSize:  Size{Height: 32},
								Font:     Font{Family: "Microsoft YaHei UI", PointSize: 9},
							},
							// 操作按钮网格
							Composite{
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
								Layout:     Grid{Columns: 2, Spacing: 12},
								Children: []Widget{
									PushButton{
										Text:      "📤 提交审批",
										MinSize:   Size{Height: 40},
										Font:      Font{Family: "Microsoft YaHei UI", PointSize: 10},
										OnClicked: guiRequestApproval,
									},
									PushButton{
										Text:      "🔄 同步状态",
										MinSize:   Size{Height: 40},
										Font:      Font{Family: "Microsoft YaHei UI", PointSize: 10},
										OnClicked: guiSyncStatus,
									},
									PushButton{
										Text:      "▶️ 执行一次",
										MinSize:   Size{Height: 40},
										Font:      Font{Family: "Microsoft YaHei UI", PointSize: 10},
										OnClicked: guiRunOnce,
									},
									PushButton{
										Text:      "⚙️ 安装守护",
										MinSize:   Size{Height: 40},
										Font:      Font{Family: "Microsoft YaHei UI", PointSize: 10},
										OnClicked: guiInstallTask,
									},
								},
							},
							PushButton{
								Text:      "🗑️ 删除计划任务",
								MinSize:   Size{Height: 36},
								Font:      Font{Family: "Microsoft YaHei UI", PointSize: 9},
								OnClicked: guiRemoveTask,
							},
						},
					},
					// 底部提示
					Composite{
						Background: SolidColorBrush{Color: walk.RGB(248, 250, 252)},
						Layout:     VBox{Spacing: 4},
						Children: []Widget{
							Label{
								Text:       "💡 首次使用请先提交审批，待管理员通过后再执行功能",
								Font:       Font{Family: "Microsoft YaHei UI", PointSize: 8},
								TextColor:  walk.RGB(100, 116, 139),
								Background: SolidColorBrush{Color: walk.RGB(248, 250, 252)},
							},
							Label{
								Text:       "⏰ 同一设备 24 小时内只允许提交一次审批申请",
								Font:       Font{Family: "Microsoft YaHei UI", PointSize: 8},
								TextColor:  walk.RGB(100, 116, 139),
								Background: SolidColorBrush{Color: walk.RGB(248, 250, 252)},
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

	// 👈 【防隐身绝杀】手动强制显示窗口！打破双击没反应的隐身错觉！
	mainWindow.SetVisible(true)
	win.ShowWindow(mainWindow.Handle(), win.SW_NORMAL)
	win.SetForegroundWindow(mainWindow.Handle())

	// 👈 【绝对防弹的加载图标】如果旁边没有合法的 app.ico，强制使用系统自带的【蓝色信息图标】，保证托盘绝对不会因为空值而消失！
	var myIcon *walk.Icon
	myIcon, _ = walk.NewIconFromFile("app.ico")
	if myIcon == nil {
		myIcon = walk.IconInformation()
	}
	mainWindow.SetIcon(myIcon)

	mainWindow.SizeChanged().Attach(func() {
		if mainWindow != nil && mainWindow.Handle() != 0 {
			if win.IsIconic(mainWindow.Handle()) {
				mainWindow.SetVisible(false)
			}
		}
	})

	var errNi error
	ni, errNi = walk.NewNotifyIcon(mainWindow)
	if errNi == nil {
		ni.SetIcon(myIcon)
		ni.SetToolTip("Fuck0Trust 守护中")

		showAction := walk.NewAction()
		showAction.SetText("显示主界面")
		showAction.Triggered().Attach(func() {
			mainWindow.SetVisible(true)
			win.ShowWindow(mainWindow.Handle(), win.SW_RESTORE)
			win.SetForegroundWindow(mainWindow.Handle())
		})
		ni.ContextMenu().Actions().Add(showAction)

		exitAction := walk.NewAction()
		exitAction.SetText("完全退出程序")
		exitAction.Triggered().Attach(func() {
			ni.Dispose()
			walk.App().Exit(0)
		})
		ni.ContextMenu().Actions().Add(exitAction)

		ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
			if button == walk.LeftButton && mainWindow != nil && mainWindow.Handle() != 0 {
				mainWindow.SetVisible(true)
				win.ShowWindow(mainWindow.Handle(), win.SW_RESTORE)
				win.SetForegroundWindow(mainWindow.Handle())
			}
		})
		ni.SetVisible(true)
	}

	mainWindow.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		mainWindow.SetVisible(false)
	})

	mainWindow.Starting().Attach(func() {
		go func() {
			time.Sleep(100 * time.Millisecond)
			initialCheck()
		}()
	})

	mainWindow.Run()
}

func initialCheck() {
	if mainWindow == nil || statusLabel == nil {
		return
	}
	mainWindow.Synchronize(func() { setStatusText("设备审批状态：同步中...", walk.RGB(71, 85, 105)) })
	if err := checkAPIReachable(8 * time.Second); err != nil {
		mainWindow.Synchronize(func() {
			setStatusText("设备审批状态：网络异常", walk.RGB(185, 28, 28))
			walk.MsgBox(mainWindow, "网络环境存在问题", "当前网络无法访问审批服务，请更换网络或检查代理后重新打开客户端。", walk.MsgBoxIconWarning)
		})
		return
	}
	status, err := refreshApprovalFromAPI(10 * time.Second)
	if err != nil {
		mainWindow.Synchronize(func() { setStatusText("设备审批状态：同步失败", walk.RGB(185, 28, 28)) })
		return
	}
	if status.Blacklisted {
		mainWindow.Synchronize(func() {
			walk.MsgBox(mainWindow, "提示", "你已被拉黑，请联系 @pppatr1ck_bot", walk.MsgBoxIconError)
			os.Exit(0)
		})
		return
	}
	mainWindow.Synchronize(func() { updateStatusLabel(status.Approved) })
}

func setStatusText(text string, color walk.Color) {
	if statusLabel != nil {
		statusLabel.SetText(text)
		statusLabel.SetTextColor(color)
	}
}

func updateStatusLabel(approved bool) {
	if approved {
		setStatusText("设备审批状态：✅ 已通过", walk.RGB(21, 128, 61))
	} else {
		setStatusText("设备审批状态：⏳ 未通过/待审批", walk.RGB(185, 28, 28))
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
				walk.MsgBox(mainWindow, "提交失败", sanitizeError(err), walk.MsgBoxIconError)
				return
			}
			walk.MsgBox(mainWindow, "提交成功", "审批请求已提交，请联系管理员审批。\n同一设备 24 小时内只能提交一次。", walk.MsgBoxIconInformation)
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
				walk.MsgBox(mainWindow, "同步失败", sanitizeError(err), walk.MsgBoxIconError)
				return
			}
			if status.Blacklisted {
				walk.MsgBox(mainWindow, "提示", "你已被拉黑，请联系 @pppatr1ck_bot", walk.MsgBoxIconError)
				os.Exit(0)
				return
			}
			updateStatusLabel(status.Approved)
			if status.Approved {
				walk.MsgBox(mainWindow, "同步成功", "✅ 当前设备审批状态：已通过", walk.MsgBoxIconInformation)
			} else {
				walk.MsgBox(mainWindow, "同步成功", "⏳ 当前设备审批状态：未通过/待审批", walk.MsgBoxIconInformation)
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
			walk.MsgBox(mainWindow, "执行成功", "✅ 受控功能已执行完成", walk.MsgBoxIconInformation)
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
				walk.MsgBox(mainWindow, "安装失败", err.Error(), walk.MsgBoxIconError)
				return
			}
			walk.MsgBox(mainWindow, "安装成功", fmt.Sprintf("✅ 计划任务已安装：%s\n守护进程已在后台启动，开机自动运行。", TaskName), walk.MsgBoxIconInformation)
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
				walk.MsgBox(mainWindow, "删除失败", err.Error(), walk.MsgBoxIconError)
				return
			}
			walk.MsgBox(mainWindow, "删除成功", fmt.Sprintf("✅ 计划任务已删除：%s", TaskName), walk.MsgBoxIconInformation)
		})
	}()
}
