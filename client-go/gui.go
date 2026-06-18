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
	"golang.org/x/sys/windows"
)

var (
	mainWindow   *walk.MainWindow
	statusLabel  *walk.Label
	noteEdit     *walk.LineEdit
	deviceIDText string
	ni           *walk.NotifyIcon
	guiMutex     windows.Handle
)

func launchGUI() {
	// 创建 GUI 互斥锁，防止多开
	mutex, err := createMutex("Global\\Fuck0TrustGUIMutex")
	if err != nil {
		walk.MsgBox(nil, "提示", "程序已在运行中，请勿重复打开。", walk.MsgBoxIconWarning)
		return
	}
	guiMutex = mutex
	defer releaseMutex(guiMutex)

	deviceIDText = deviceID()
	shortDeviceID := deviceIDText
	if len(deviceIDText) > 32 {
		shortDeviceID = deviceIDText[:16] + "..." + deviceIDText[len(deviceIDText)-8:]
	}

	if err := (MainWindow{
		AssignTo:   &mainWindow,
		Title:      "Fuck0Trust",
		Size:       Size{Width: 340, Height: 420},
		MinSize:    Size{Width: 340, Height: 420},
		MaxSize:    Size{Width: 340, Height: 420},
		Layout:     VBox{MarginsZero: true, SpacingZero: true},
		Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},

		Children: []Widget{
			// 顶部标题栏 - 现代化设计
			Composite{
				Background: SolidColorBrush{Color: walk.RGB(99, 102, 241)}, // 紫蓝色
				MinSize:    Size{Height: 60},
				MaxSize:    Size{Height: 60},
				Layout:     VBox{Margins: Margins{Left: 16, Top: 14, Right: 16, Bottom: 14}},
				Children: []Widget{
					Label{
						Text:       "🛡️ Fuck0Trust",
						Font:       Font{Family: "Microsoft YaHei UI", PointSize: 16, Bold: true},
						TextColor:  walk.RGB(255, 255, 255),
						Background: SolidColorBrush{Color: walk.RGB(99, 102, 241)},
					},
				},
			},
			// 主内容区域
			Composite{
				Background: SolidColorBrush{Color: walk.RGB(249, 250, 251)},
				Layout:     VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 16}, Spacing: 12},
				Children: []Widget{
					// 状态卡片 - 简洁设计
					Composite{
						Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
						Layout:     VBox{Margins: Margins{Left: 16, Top: 12, Right: 16, Bottom: 12}, Spacing: 8},
						Children: []Widget{
							Label{
								AssignTo:   &statusLabel,
								Text:       "设备状态：检测中...",
								Font:       Font{Family: "Microsoft YaHei UI", PointSize: 10, Bold: true},
								TextColor:  walk.RGB(17, 24, 39),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
								MinSize:    Size{Height: 22},
							},
							Composite{
								Background: SolidColorBrush{Color: walk.RGB(243, 244, 246)},
								Layout:     VBox{Margins: Margins{Left: 10, Top: 6, Right: 10, Bottom: 6}},
								MinSize:    Size{Height: 26},
								Children: []Widget{
									Label{
										Text:       "设备 ID: " + shortDeviceID,
										Font:       Font{Family: "Consolas", PointSize: 8},
										TextColor:  walk.RGB(107, 114, 128),
										Background: SolidColorBrush{Color: walk.RGB(243, 244, 246)},
									},
								},
							},
						},
					},
					// 审批申请区域 - 卡片设计
					Composite{
						Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
						Layout:     VBox{Margins: Margins{Left: 16, Top: 12, Right: 16, Bottom: 12}, Spacing: 10},
						Children: []Widget{
							Label{
								Text:       "📝 联系方式或申请理由",
								Font:       Font{Family: "Microsoft YaHei UI", PointSize: 9, Bold: true},
								TextColor:  walk.RGB(17, 24, 39),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
							},
							LineEdit{
								AssignTo: &noteEdit,
								MinSize:  Size{Height: 28},
								Font:     Font{Family: "Microsoft YaHei UI", PointSize: 9},
							},
							// 操作按钮网格
							Composite{
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
								Layout:     Grid{Columns: 2, Spacing: 8},
								Children: []Widget{
									PushButton{
										Text:      "📤 提交审批",
										MinSize:   Size{Height: 36},
										Font:      Font{Family: "Microsoft YaHei UI", PointSize: 9},
										OnClicked: guiRequestApproval,
									},
									PushButton{
										Text:      "🔄 同步状态",
										MinSize:   Size{Height: 36},
										Font:      Font{Family: "Microsoft YaHei UI", PointSize: 9},
										OnClicked: guiSyncStatus,
									},
									PushButton{
										Text:      "⚙️ 安装守护",
										MinSize:   Size{Height: 36},
										Font:      Font{Family: "Microsoft YaHei UI", PointSize: 9},
										OnClicked: guiInstallTask,
									},
									PushButton{
										Text:      "🗑️ 删除退出",
										MinSize:   Size{Height: 36},
										Font:      Font{Family: "Microsoft YaHei UI", PointSize: 9},
										OnClicked: guiRemoveTask,
									},
								},
							},
						},
					},
					// 底部提示 - 醒目红色
					Composite{
						Background: SolidColorBrush{Color: walk.RGB(249, 250, 251)},
						Layout:     VBox{Spacing: 4},
						Children: []Widget{
							Label{
								Text:       "💡 首次使用请先提交审批，待管理员通过后再操作",
								Font:       Font{Family: "Microsoft YaHei UI", PointSize: 8},
								TextColor:  walk.RGB(239, 68, 68),
								Background: SolidColorBrush{Color: walk.RGB(249, 250, 251)},
							},
							Label{
								Text:       "⏰ 24 小时内仅可提交一次审批申请",
								Font:       Font{Family: "Microsoft YaHei UI", PointSize: 8},
								TextColor:  walk.RGB(239, 68, 68),
								Background: SolidColorBrush{Color: walk.RGB(249, 250, 251)},
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

	// 禁用最大化按钮和调整大小
	hwnd := mainWindow.Handle()
	style := win.GetWindowLong(hwnd, win.GWL_STYLE)
	style &^= win.WS_MAXIMIZEBOX | win.WS_SIZEBOX
	win.SetWindowLong(hwnd, win.GWL_STYLE, style)

	// 👈 【防隐身绝杀】手动强制显示窗口！打破双击没反应的隐身错觉！
	mainWindow.SetVisible(true)
	win.ShowWindow(mainWindow.Handle(), win.SW_NORMAL)
	win.SetForegroundWindow(mainWindow.Handle())

	// 从嵌入资源加载图标（rsrc 将 app.ico 嵌入到 exe）
	var myIcon *walk.Icon

	// 尝试常见的图标资源 ID（rsrc 可能使用不同的 ID）
	iconIDs := []uintptr{1, 2, 3, 101, 32512}
	for _, id := range iconIDs {
		if icon, err := walk.NewIconFromResourceId(int(id)); err == nil && icon != nil {
			myIcon = icon
			break
		}
	}

	// 如果所有 ID 都失败，使用系统图标
	if myIcon == nil {
		myIcon = walk.IconShield()
	}

	mainWindow.SetIcon(myIcon)

	mainWindow.SizeChanged().Attach(func() {
		if mainWindow != nil && mainWindow.Handle() != 0 {
			if win.IsIconic(mainWindow.Handle()) {
				// 最小化时隐藏到托盘
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

	// 窗口关闭事件：点击 X 直接退出 GUI
	mainWindow.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		// 不取消关闭，直接退出
		if ni != nil {
			ni.Dispose()
		}
		walk.App().Exit(0)
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
	mainWindow.Synchronize(func() { setStatusText("设备状态：同步中...", walk.RGB(71, 85, 105)) })
	if err := checkAPIReachable(8 * time.Second); err != nil {
		mainWindow.Synchronize(func() {
			setStatusText("设备状态：网络异常", walk.RGB(185, 28, 28))
			walk.MsgBox(mainWindow, "网络环境存在问题", "当前网络无法访问审批服务，请更换网络或检查代理后重新打开客户端。", walk.MsgBoxIconWarning)
		})
		return
	}
	status, err := refreshApprovalFromAPI(10 * time.Second)
	if err != nil {
		mainWindow.Synchronize(func() { setStatusText("设备状态：同步失败", walk.RGB(185, 28, 28)) })
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
		setStatusText("设备状态：✅ 已通过", walk.RGB(21, 128, 61))
	} else {
		setStatusText("设备状态：⏳ 待审批", walk.RGB(185, 28, 28))
	}
}

func guiRequestApproval() {
	if noteEdit == nil {
		return
	}
	note := strings.TrimSpace(noteEdit.Text())
	if note == "" {
		walk.MsgBox(mainWindow, "提示", "请填写你的可联系方式或申请理由，否则不予通过", walk.MsgBoxIconWarning)
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
				walk.MsgBox(mainWindow, "同步成功", "✅ 审批已通过", walk.MsgBoxIconInformation)
			} else {
				walk.MsgBox(mainWindow, "同步成功", "⏳ 待审批", walk.MsgBoxIconInformation)
			}
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
			walk.MsgBox(mainWindow, "安装成功", fmt.Sprintf("✅ 任务：%s\n守护已启动", TaskName), walk.MsgBoxIconInformation)
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
			walk.MsgBox(mainWindow, "删除成功", fmt.Sprintf("计划任务已删除或不存在：%s\n程序即将退出。", TaskName), walk.MsgBoxIconInformation)
			if ni != nil {
				ni.Dispose()
			}
			walk.App().Exit(0)
		})
	}()
}
