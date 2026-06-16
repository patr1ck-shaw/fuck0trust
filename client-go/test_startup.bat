@echo off
chcp 65001 >nul
echo ================================
echo Fuck0Trust 客户端启动测试
echo ================================
echo.

set EXE=Fuck0TrustClient.exe
set LOG=startup_test.log

if not exist "%EXE%" (
    echo [错误] 找不到 %EXE%
    echo 请先从 GitHub Actions 下载构建好的可执行文件
    pause
    exit /b 1
)

echo [信息] 找到可执行文件: %EXE%
echo [信息] 开始启动测试...
echo [信息] 日志将保存到: %LOG%
echo.

echo ======== 启动测试开始于 %date% %time% ======== > "%LOG%"
echo. >> "%LOG%"

echo [测试1] 无参数启动 (GUI 模式)
echo [测试1] 无参数启动 (GUI 模式) >> "%LOG%"
"%EXE%" >> "%LOG%" 2>&1
set EXIT_CODE=%ERRORLEVEL%
echo 退出码: %EXIT_CODE% >> "%LOG%"
echo. >> "%LOG%"

if %EXIT_CODE% EQU 0 (
    echo [成功] 程序正常退出
) else (
    echo [失败] 程序异常退出，退出码: %EXIT_CODE%
)
echo.

echo [测试2] 查看帮助信息
echo [测试2] 查看帮助信息 >> "%LOG%"
"%EXE%" help >> "%LOG%" 2>&1
echo. >> "%LOG%"

echo [测试3] 查询审批状态
echo [测试3] 查询审批状态 >> "%LOG%"
"%EXE%" status >> "%LOG%" 2>&1
echo. >> "%LOG%"

echo.
echo ================================
echo 测试完成！
echo ================================
echo 请查看 %LOG% 了解详细日志
echo.
echo 如果程序闪退，崩溃日志可能保存在以下位置：
echo %TEMP%\fuck0trust_crash.log
echo %TEMP%\fuck0trust_startup.log
echo %TEMP%\fuck0trust_deviceid_error.log
echo.

if exist "%TEMP%\fuck0trust_crash.log" (
    echo [发现崩溃日志] %TEMP%\fuck0trust_crash.log
    type "%TEMP%\fuck0trust_crash.log"
    echo.
)

if exist "%TEMP%\fuck0trust_startup.log" (
    echo [发现启动日志] %TEMP%\fuck0trust_startup.log
    type "%TEMP%\fuck0trust_startup.log"
    echo.
)

pause
