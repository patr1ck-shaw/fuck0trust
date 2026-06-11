//go:build tray && !desktop && !walkgui
// +build tray,!desktop,!walkgui

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/getlantern/systray"
)

func launchTray() {
	systray.Run(onReady, onExit)
}

func onReady() {
	// 设置托盘图标和提示
	systray.SetTitle("Fuck0Trust")
	systray.SetTooltip("Fuck0Trust")


	
	// 创建菜单项
	mStatus := systray.AddMenuItem("状态：检测中...", "当前审批状态")
	mStatus.Disable()
	
	systray.AddSeparator()
	
	mDeviceID := systray.AddMenuItem("查看设备 ID", "显示设备标识")
	mRequestApproval := systray.AddMenuItem("提交审批", "提交设备审批申请")
	mSyncStatus := systray.AddMenuItem("同步状态", "同步审批状态")
	
	systray.AddSeparator()
	
	mRunOnce := systray.AddMenuItem("执行一次", "立即执行功能")
	mInstallTask := systray.AddMenuItem("安装计划任务", "安装定时任务")
	mRemoveTask := systray.AddMenuItem("删除计划任务", "删除定时任务")
	
	systray.AddSeparator()
	
	mQuit := systray.AddMenuItem("退出", "退出程序")
	
	// 初始检查状态
	go func() {
		// 检查网络
		if err := checkAPIReachable(8 * time.Second); err != nil {
			mStatus.SetTitle("状态：网络异常")
			showNotification("网络异常", "无法连接到审批服务器")
			return
		}
		
		// 获取审批状态
		status, err := refreshApprovalFromAPI(10 * time.Second)
		if err != nil {
			mStatus.SetTitle("状态：同步失败")
			return
		}
		
		if status.Blacklisted {
			showNotification("提示", "你已被拉黑，请联系 @pppatr1ck_bot")
			time.Sleep(3 * time.Second)
			systray.Quit()
			os.Exit(0)
			return
		}
		if status.Approved {
			mStatus.SetTitle("状态：已通过 ✓")
		} else {
			mStatus.SetTitle("状态：未通过/待审批")
		}
	}()
	
	// 事件循环
	go func() {
		for {
			select {
			case <-mDeviceID.ClickedCh:
				deviceIDText := deviceID()
				showNotification("设备 ID", deviceIDText)
				
			case <-mRequestApproval.ClickedCh:
				go func() {
					if err := requestApproval(""); err != nil {
						showNotification("提交失败", sanitizeError(err))
					} else {

						showNotification("提交成功", "已提交待管理员审批")
						// 刷新状态
						if status, err := refreshApprovalFromAPI(10 * time.Second); err == nil {
							if status.Approved {
								mStatus.SetTitle("状态：已通过 ✓")
							} else {
								mStatus.SetTitle("状态：未通过/待审批")
							}
						}
					}
				}()
				
			case <-mSyncStatus.ClickedCh:
				go func() {
					status, err := refreshApprovalFromAPI(10 * time.Second)
					if err != nil {
						showNotification("同步失败", sanitizeError(err))
						return
					}

					if status.Blacklisted {
						showNotification("提示", "你已被拉黑，请联系 @pppatr1ck_bot")
						time.Sleep(3 * time.Second)
						systray.Quit()
						os.Exit(0)
						return
					}
					if status.Approved {
						mStatus.SetTitle("状态：已通过 ✓")
						showNotification("审批状态", "当前设备已通过审批")
					} else {
						mStatus.SetTitle("状态：未通过/待审批")
						showNotification("审批状态", "当前设备未通过或待审批")
					}
				}()
				
			case <-mRunOnce.ClickedCh:
				go func() {
					if err := runOnce(); err != nil {
						showNotification("执行失败", err.Error())
					} else {
						showNotification("执行完成", "受控功能已执行")
					}
				}()
				
			case <-mInstallTask.ClickedCh:
				go func() {
					if err := installTask(); err != nil {
						showNotification("安装失败", err.Error())
					} else {
						showNotification("安装完成", fmt.Sprintf("计划任务已创建：%s", TaskName))
					}
				}()
				
			case <-mRemoveTask.ClickedCh:
				go func() {
					if err := removeTask(); err != nil {
						showNotification("删除失败", err.Error())
					} else {
						showNotification("删除完成", fmt.Sprintf("计划任务已删除：%s", TaskName))
					}
				}()
				
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// 清理工作
}

// Windows 通知
func showNotification(title, message string) {
	// 使用 PowerShell 显示 Windows 通知
	script := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

$template = @"
<toast>
    <visual>
        <binding template="ToastText02">
            <text id="1">%s</text>
            <text id="2">%s</text>
        </binding>
    </visual>
</toast>
"@

$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("Fuck0Trust").Show($toast)

`, title, message)
	
	// 后台执行，不阻塞
	go func() {
		tmpFile := filepath.Join(os.TempDir(), "fuck0trust_notify.ps1")
		os.WriteFile(tmpFile, []byte(script), 0644)
		exec.Command("powershell", "-WindowStyle", "Hidden", "-File", tmpFile).Run()
		os.Remove(tmpFile)
	}()
}
