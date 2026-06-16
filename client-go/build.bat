@echo off
setlocal enabledelayedexpansion

echo ========================================
echo  Fuck0Trust Client - Go Build Script
echo ========================================
echo.

REM 检查 Go 是否安装
where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Go 未安装或不在 PATH 中，请先安装 Go
    exit /b 1
)

echo [1/6] 清理旧文件...
if exist "Fuck0TrustClient.exe" del /f /q "Fuck0TrustClient.exe"
if exist "rsrc.syso" del /f /q "rsrc.syso"

echo [2/6] 下载依赖...
go mod download
if %ERRORLEVEL% neq 0 (
    echo [ERROR] 依赖下载失败
    exit /b 1
)

echo [3/6] 整理依赖...
go mod tidy
if %ERRORLEVEL% neq 0 (
    echo [ERROR] 依赖整理失败
    exit /b 1
)

echo [4/6] 生成资源文件（嵌入 manifest 和图标）...
REM 如果有 rsrc 工具，使用它生成 .syso 文件
where rsrc >nul 2>&1
if %ERRORLEVEL% equ 0 (
    rsrc -manifest fuck0trust.manifest -ico app.ico -o rsrc.syso
    if %ERRORLEVEL% equ 0 (
        echo [OK] 资源文件生成成功（包含图标）
    ) else (
        echo [WARN] 资源文件生成失败，尝试仅嵌入 manifest
        rsrc -manifest fuck0trust.manifest -o rsrc.syso
    )
) else (
    echo [WARN] rsrc 工具未安装，跳过 manifest 和图标嵌入
    echo [INFO] 安装方法：go install github.com/akavel/rsrc@latest
)

echo [5/6] 编译程序（使用 walkgui 标签）...
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64
go build -tags walkgui -ldflags="-H=windowsgui" -o Fuck0TrustClient.exe .
if %ERRORLEVEL% neq 0 (
    echo [ERROR] 编译失败
    exit /b 1
)

echo [6/6] 压缩可执行文件（可选）...
where upx >nul 2>&1
if %ERRORLEVEL% equ 0 (
    echo [INFO] 使用 UPX 压缩...
    upx --best --lzma Fuck0TrustClient.exe 2>nul
    if %ERRORLEVEL% equ 0 (
        echo [OK] 压缩成功
    ) else (
        echo [WARN] 压缩失败，但不影响使用
    )
) else (
    echo [INFO] UPX 未安装，跳过压缩
    echo [INFO] 可选安装：choco install upx
)

echo.
echo ========================================
echo  构建完成！
echo ========================================
echo.
echo 输出文件: Fuck0TrustClient.exe
dir Fuck0TrustClient.exe | findstr "exe"
echo.
echo 使用说明：
echo   - GUI 模式：双击运行或执行 Fuck0TrustClient.exe
echo   - CLI 模式：Fuck0TrustClient.exe [命令]
echo.
echo 可用命令：
echo   request --note "备注"  提交审批申请
echo   status                 查询审批状态
echo   run                    执行一次受控功能
echo   guard                  启动守护进程（NetCheck 模式）
echo   install-task           安装计划任务
echo   remove-task            删除计划任务
echo.

endlocal
