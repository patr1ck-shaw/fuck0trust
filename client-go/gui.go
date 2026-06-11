//go:build walkgui
// +build walkgui

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)


var (
	mainWindow   *walk.MainWindow
	statusLabel  *walk.Label
	noteEdit     *walk.LineEdit
	deviceIDText string
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
	
	// 先测试最小化窗口能否创建
	if err := testMinimalWindow(); err != nil {
		if f, err2 := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err2 == nil {
			fmt.Fprintf(f, "Minimal window test FAILED, aborting: %v\n", err)
			f.Close()
		}
		walk.MsgBox(nil, "错误", "窗口系统初始化失败: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	
	deviceIDText = deviceID()
	
	// 记录成功获取 deviceID
	if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "Step 2: Device ID obtained: %s\n", deviceIDText)
		f.Close()
	}
	
	shortDeviceID := deviceIDText
	if len(deviceIDText) > 32 {
		shortDeviceID = deviceIDText[:16] + "..." + deviceIDText[len(deviceIDText)-8:]
	}
	
	// 记录开始创建窗口
	if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "Step 3: Creating main window...\n")
		f.Close()
	}
	
	if err := (MainWindow{
		AssignTo: &mainWindow,
		Title:    "Fuck0Trust 审批客户端",
		MinSize:  Size{Width: 560, Height: 420},
		MaxSize:  Size{Width: 560, Height: 420},
		Layout:   VBox{MarginsZero: true, SpacingZero: true},
		Background: SolidColorBrush{Color: walk.RGB(246, 247, 251)},
		Children: []Widget{
			// 顶部蓝色标题栏
			Composite{
				Background: SolidColorBrush{Color: walk.RGB(37, 99, 235)},
				MinSize:    Size{Height: 82},
				MaxSize:    Size{Height: 82},
				Layout:     VBox{},
				Children: []Widget{
					Label{
						Text:       "Fuck0Trust 审批客户端",
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
							// 状态标签
							Label{
								AssignTo:   &statusLabel,
								Text:       "当前设备审批状态：检测中",
								Font:       Font{Family: "Microsoft YaHei", PointSize: 11, Bold: true},
								TextColor:  walk.RGB(51, 65, 85),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
								MinSize:    Size{Height: 30},
							},
							// 设备 ID
							Label{
								Text:       "设备 ID：" + shortDeviceID,
								Font:       Font{Family: "Microsoft YaHei", PointSize: 9},
								TextColor:  walk.RGB(100, 116, 139),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
								MinSize:    Size{Height: 20},
							},
							VSpacer{Size: 8},
							// 申请备注标签
							Label{
								Text:       "申请备注（可选）：",
								Font:       Font{Family: "Microsoft YaHei", PointSize: 9},
								TextColor:  walk.RGB(51, 65, 85),
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
							},
							// 备注输入框
							LineEdit{
								AssignTo: &noteEdit,
								MaxSize:  Size{Height: 26},
							},
							VSpacer{Size: 10},
							// 按钮区域 - 3列2行布局
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
									// 占位符保持对齐
									Composite{
										Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
									},
								},
							},
							VSpacer{Size: 8},
							// 底部说明文字
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
		// 记录窗口创建失败
		if f, err2 := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err2 == nil {
			fmt.Fprintf(f, "Step 4: FAILED to create window: %v\n", err)
			f.Close()
		}
		walk.MsgBox(nil, "错误", "创建窗口失败: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	
	// 记录窗口创建成功
	if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "Step 4: Window created successfully\n")
		fmt.Fprintf(f, "Step 5: Starting main window event loop...\n")
		f.Close()
	}

	// 延迟 300ms 后执行初始检查
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

	// 先检查网络连通性
	if err := checkAPIReachable(8 * time.Second); err != nil {
		mainWindow.Synchronize(func() {
			setStatusText("当前设备审批状态：网络异常", walk.RGB(146, 64, 14))
			walk.MsgBox(mainWindow, "网络环境存在问题", 
				"当前网络无法访问审批服务，请更换网络或检查代理后重新打开客户端。", 
				walk.MsgBoxIconWarning)
		})
		return
	}

	// 刷新审批状态
	status, err := refreshApprovalFromAPI(10 * time.Second)
	if err != nil {
		mainWindow.Synchronize(func() {
			setStatusText("当前设备审批状态：同步失败", walk.RGB(146, 64, 14))
		})
		return
	}

	mainWindow.Synchronize(func() {
		updateStatusLabel(status.Approved)
	})
}

// 设置状态文本和颜色
func setStatusText(text string, color walk.Color) {
	statusLabel.SetText(text)
	statusLabel.SetTextColor(color)
}

// 更新状态标签
func updateStatusLabel(approved bool) {
	if approved {
		setStatusText("当前设备审批状态：已通过", walk.RGB(22, 101, 52))
	} else {
		setStatusText("当前设备审批状态：未通过/待审批", walk.RGB(153, 27, 27))
	}
}

// GUI - 提交审批
func guiRequestApproval() {
	note := noteEdit.Text()
	
	// 在后台线程执行
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

// GUI - 同步审批状态
func guiSyncStatus() {
	go func() {
		status, err := refreshApprovalFromAPI(10 * time.Second)
		mainWindow.Synchronize(func() {
			if err != nil {
				walk.MsgBox(mainWindow, "执行失败", sanitizeError(err), walk.MsgBoxIconError)
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

// GUI - 执行一次
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

// GUI - 安装计划任务
func guiInstallTask() {
	go func() {
		err := installTask()
		mainWindow.Synchronize(func() {
			if err != nil {
				walk.MsgBox(mainWindow, "执行失败", err.Error(), walk.MsgBoxIconError)
				return
			}
			walk.MsgBox(mainWindow, "安装完成", 
				fmt.Sprintf("计划任务已创建/更新：%s，每 4 分钟执行一次。", TaskName), 
				walk.MsgBoxIconInformation)
		})
	}()
}

// GUI - 删除计划任务
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



