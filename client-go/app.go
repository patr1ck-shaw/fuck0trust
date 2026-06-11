package main

import (
	"context"
	"fmt"
	"time"
)

// App 应用结构
type App struct {
	ctx context.Context
}

// NewApp 创建应用实例
func NewApp() *App {
	return &App{}
}

// startup 在应用启动时调用
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetDeviceID 获取设备ID
func (a *App) GetDeviceID() string {
	return deviceID()
}

// GetApprovalStatus 获取审批状态
func (a *App) GetApprovalStatus() map[string]interface{} {
	isApproved := isLocallyApproved()
	remaining := secondsUntilNextRequest()
	
	result := map[string]interface{}{
		"approved":  isApproved,
		"deviceId":  deviceID(),
		"canSubmit": remaining == 0,
	}
	
	if remaining > 0 {
		result["nextRequestIn"] = formatDuration(remaining)
	}
	
	return result
}

// SubmitApproval 提交审批请求
func (a *App) SubmitApproval(note string) map[string]interface{} {
	err := requestApproval(note)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}
	return map[string]interface{}{
		"success": true,
		"message": "审批请求已提交，请联系管理员审批",
	}
}

// SyncStatus 同步审批状态
func (a *App) SyncStatus() map[string]interface{} {
	status, err := refreshApprovalFromAPI(20 * time.Second)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": friendlyNetworkError(),
		}
	}
	
	return map[string]interface{}{
		"success":  true,
		"approved": status.Approved,
		"message":  "状态已同步",
	}
}

// RunOnce 执行一次受控功能
func (a *App) RunOnce() map[string]interface{} {
	err := runOnce()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}
	return map[string]interface{}{
		"success": true,
		"message": "执行成功",
	}
}

// InstallTask 安装计划任务
func (a *App) InstallTask() map[string]interface{} {
	err := installTask()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}
	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("计划任务已创建: %s", TaskName),
	}
}

// RemoveTask 删除计划任务
func (a *App) RemoveTask() map[string]interface{} {
	err := removeTask()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
		}
	}
	return map[string]interface{}{
		"success": true,
		"message": "计划任务已删除",
	}
}
