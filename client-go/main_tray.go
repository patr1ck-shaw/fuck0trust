//go:build tray && !desktop && !walkgui
// +build tray,!desktop,!walkgui

package main

func launchGUI() {
	launchTray()
}
