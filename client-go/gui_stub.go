//go:build !walkgui
// +build !walkgui

package main

import (
	"fmt"
	"os"
)

// launchGUI 在非 GUI 模式下的占位实现
func launchGUI() {
	fmt.Fprintln(os.Stderr, "GUI 模式未启用，请使用命令行参数运行")
	fmt.Println(`用法: Fuck0TrustClient.exe [命令]`)
	fmt.Println(`命令:`)
	fmt.Println(`  request [--note 备注]  - 提交审批申请`)
	fmt.Println(`  status                  - 查询审批状态`)
	fmt.Println(`  run                     - 执行一次受控功能`)
	fmt.Println(`  guard                   - 启动守护进程（NetCheck 模式）`)
	fmt.Println(`  install-task            - 安装计划任务`)
	fmt.Println(`  remove-task             - 删除计划任务`)
	os.Exit(1)
}
