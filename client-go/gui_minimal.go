package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// 测试最小化窗口
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
	
	err := MainWindow{
		AssignTo: &mw,
		Title:    "Test",
		Size:     Size{Width: 300, Height: 200},
	}.Create()
	
	if err != nil {
		logStep(fmt.Sprintf("Minimal window creation failed: %v", err))
		return err
	}
	
	logStep("Minimal window created successfully!")
	
	// 立即关闭
	mw.Close()
	return nil
}
