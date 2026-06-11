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

echo [1/5] 清理旧文件...
if exist "Fuck0TrustClient.exe" del /f /q "Fuck0TrustClient.exe"
if exist "rsrc.syso" del /f /q "rsrc.syso"

echo [2/5] 下载依赖...
go mod download
if %ERRORLEVEL% neq 0 (
    echo [ERROR] 依赖下载失败
    exit /b 1
)

echo [3/5] 生成资源文件（嵌入 manifest）...
REM 如果有 rsrc 工具，使用它生成 .syso 文件
where rsrc >nul 2>&1
if %ERRORLEVEL% equ 0 (
    rsrc -manifest fuck0trust.manifest -o rsrc.syso
) else (
    echo [WARN] rsrc 工具未安装，跳过 manifest 嵌入
    echo [INFO] 可选：安装 rsrc - go install github.com/akavel/rsrc@latest
)

echo [4/5] 编译程序...
go build -ldflags="-s -w -H=windowsgui" -o Fuck0TrustClient.exe .
if %ERRORLEVEL% neq 0 (
    echo [ERROR] 编译失败
    exit /b 1
)

echo [5/5] 压缩可执行文件（可选）...
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
    echo [INFO] UPX 未安装，跳过压缩（可选优化）
)

echo.
echo ========================================
echo  构建完成！
echo ========================================
echo.
echo 输出文件: Fuck0TrustClient.exe
dir Fuck0TrustClient.exe | findstr "exe"
echo.
echo 测试运行: Fuck0TrustClient.exe
echo CLI 模式: Fuck0TrustClient.exe status
echo.

endlocal
