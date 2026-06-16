@echo off
echo ========================================
echo 简化测试脚本
echo ========================================
echo.

echo [1] 检查 exe 文件
if exist "Fuck0TrustClient.exe" (
    echo     找到: Fuck0TrustClient.exe
) else (
    echo     错误: 找不到 Fuck0TrustClient.exe
    echo     当前目录: %CD%
    dir *.exe
    pause
    exit /b 1
)

echo.
echo [2] 直接运行 exe (会闪退的话继续下一步)
echo     按任意键开始运行...
pause >nul
start /wait Fuck0TrustClient.exe
echo     程序已退出，退出码: %ERRORLEVEL%

echo.
echo [3] 运行并捕获输出到文件
Fuck0TrustClient.exe > output.txt 2>&1
echo     退出码: %ERRORLEVEL%
echo     输出已保存到 output.txt

echo.
echo [4] 显示输出内容
type output.txt

echo.
echo [5] 检查临时日志文件
echo     临时目录: %TEMP%
echo.
if exist "%TEMP%\fuck0trust_startup.log" (
    echo     发现启动日志:
    type "%TEMP%\fuck0trust_startup.log"
) else (
    echo     未找到启动日志
)

echo.
if exist "%TEMP%\fuck0trust_crash.log" (
    echo     发现崩溃日志:
    type "%TEMP%\fuck0trust_crash.log"
) else (
    echo     未找到崩溃日志
)

echo.
echo ========================================
echo 测试完成！请将上述内容截图或复制给我
echo ========================================
pause
