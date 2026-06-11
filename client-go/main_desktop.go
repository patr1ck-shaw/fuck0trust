//go:build desktop || (!tray && !walkgui)
// +build desktop !tray,!walkgui

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func launchGUI() {
	defer func() {
		if r := recover(); r != nil {
			writeStartupLog(fmt.Sprintf("Wails panic: %v\n%s", r, string(debug.Stack())))
			fmt.Fprintf(os.Stderr, "启动失败: %v\n", r)
		}
	}()

	writeStartupLog("Starting Wails GUI")
	app := NewApp()
	
	err := wails.Run(&options.App{
		Title:  "Fuck0Trust 审批客户端",
		Width:  600,
		Height: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})
	
	if err != nil {
		writeStartupLog(fmt.Sprintf("Wails run failed: %v", err))
		fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
		os.Exit(1)
	}
}

func writeStartupLog(message string) {
	logFile := filepath.Join(os.TempDir(), "fuck0trust_wails_startup.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), message)
}
