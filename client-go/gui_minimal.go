//go:build walkgui
// +build walkgui

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// 测试最小化窗口环境
func testMinimalWindow() error {
	logFile := filepath.Join(os.TempDir(), "fuck0trust_startup.log")
	
	logStep := func(step string) {
		if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(f, "%s\n", step)
			f.Close()
		}
	}
	
	logStep("Testing minimal window creation...")
	
	var mw *walk.MainWindow
	
	// 👈 【核心修复】为测试窗口添加 VBox 基础布局排列，彻底拦截 `CreateLayoutItem` 空指针崩溃
	err := MainWindow{
		AssignTo: &mw,
		Title:    "Test",
		Size:     Size{Width: 300, Height: 200},
		Layout:   VBox{MarginsZero: true, SpacingZero: true}, // 👈 焊死此处的空指针死穴
	}.Create()
	
	if err != nil {
		logStep(fmt.Sprintf("Minimal window creation failed: %v", err))
		return err
	}
	
	logStep("Minimal window created successfully!")
	
	// 立即关闭释放环境
	mw.Close()
	return nil
}