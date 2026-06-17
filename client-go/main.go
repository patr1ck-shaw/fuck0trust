package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/debug" // 👈 【已补上】用于打印崩溃时的精准红字代码行数
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// 👈 【核心修复】移除了原本卡死二进制文件头的 var assets embed.FS 残留，彻底解决 DEP 拦截导致的秒闪退

const (
	AppName                 = "Fuck0TrustApprovalClient"
	TaskName                = "Fuck0Trust_Status_Check"
	ServiceName             = "WFPRedirect"
	APIBase                 = "https://00.cn01.eu.cc"     // 主 API 地址
	APIBaseFallback         = "https://0.cn01.eu.cc"      // 兜底 API 地址
	RequestIntervalSeconds  = 24 * 60 * 60
	DefaultConnectTimeout   = 8 * time.Second
	DefaultReadTimeout      = 25 * time.Second
	GuardMutexName          = "Global\\Fuck0TrustGuardMutex"    // 守护进程互斥锁名称

	// NetCheck 守护进程配置
	SDPProcessName          = "sdp.exe"                          // 监测的目标进程名
	MSFTConnectTestURL      = "http://www.msftconnecttest.com/connecttest.txt" // 微软连接测试 URL
	MSFTConnectTestKeyword  = "Microsoft Connect Test"          // 预期响应内容
	GuardLoopInterval       = 5 * time.Second                   // 守护循环间隔（对齐 NetCheck.bat）
	ProcessCheckWaitTime    = 10 * time.Second                  // sdp.exe 不存在时的等待时间
	FixCooldownTime         = 3 * time.Second                   // 修复操作后的冷却时间
	StatusCheckInterval     = 24 * time.Hour                    // 定期校验审批状态间隔
)

var (
	configDir  string
	configFile string
	logFile    string  // 守护进程日志文件路径
	// 👈 【优化】关闭全局可能由于抢跑引发 nil 指针崩溃的全局变量，全面切到局部的封闭式 http 客户端
)

func init() {
	// 初始化配置目录
	programData := os.Getenv("PROGRAMDATA")
	if programData == "" {
		home, _ := os.UserHomeDir()
		programData = home
	}
	configDir = filepath.Join(programData, AppName)
	configFile = filepath.Join(configDir, "config.json")
	logFile = filepath.Join(configDir, "guard_log.txt")
}

// 获取机器 GUID
func machineGUID() string {
	defer func() {
		if r := recover(); r != nil {
			// 如果获取 GUID 出现 panic，返回默认值
		}
	}()
	
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err != nil {
		return getMACAddress()
	}
	defer k.Close()

	guid, _, err := k.GetStringValue("MachineGuid")
	if err != nil {
		return getMACAddress()
	}
	return guid
}

// 获取 MAC 地址作为备用
func getMACAddress() string {
	defer func() {
		if r := recover(); r != nil {
			// 如果获取 MAC 出现 panic，返回默认值
		}
	}()
	
	out, err := exec.Command("getmac", "/fo", "csv", "/nh").Output()
	if err != nil {
		return "unknown"
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		parts := strings.Split(lines[0], ",")
		if len(parts) > 0 {
			mac := strings.Trim(parts[0], "\" \r\n")
			if mac != "" {
				return mac
			}
		}
	}
	return "unknown"
}

// 生成设备 ID
func deviceID() string {
	defer func() {
		if r := recover(); r != nil {
			// 记录 panic 但不中断程序
			logFile := filepath.Join(os.TempDir(), "fuck0trust_deviceid_error.log")
			f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				fmt.Fprintf(f, "\n=== DeviceID Error at %s ===\n", time.Now().Format("2006-01-02 15:04:05"))
				fmt.Fprintf(f, "Panic: %v\n", r)
				f.Close()
			}
		}
	}()
	
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown-host"
	}
	
	currentUser, _ := user.Current()
	username := "unknown-user"
	if currentUser != nil && currentUser.Username != "" {
		// 规范化用户名：移除域名前缀 (DOMAIN\user -> user)
		username = currentUser.Username
		if idx := strings.LastIndex(username, "\\"); idx >= 0 {
			username = username[idx+1:]
		}
	}
	
	guid := machineGUID()
	if guid == "" {
		guid = "unknown-guid"
	}

	raw := strings.Join([]string{
		hostname,
		runtime.GOOS,
		runtime.GOARCH,
		guid,
		username,
	}, "|")

	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash)
}

// 配置结构
type Config map[string]interface{}

// 加载配置
func loadConfig() Config {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return make(Config)
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return make(Config)
	}
	return config
}

// 保存配置
func saveConfig(config Config) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	// 清理无用的旧版本字段
	delete(config, "api")

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

// 审批缓存键
func approvalCacheKey() string {
	return fmt.Sprintf("approval:%s", deviceID())
}

// 请求缓存键
func requestCacheKey() string {
	return fmt.Sprintf("request:%s", deviceID())
}

// 最后校验时间缓存键
func lastCheckCacheKey() string {
	return fmt.Sprintf("lastCheck:%s", deviceID())
}

// 保存最后校验时间
func saveLastCheckTime() error {
	config := loadConfig()
	config[lastCheckCacheKey()] = time.Now().Unix()
	return saveConfig(config)
}

// 获取距离上次校验的时长
func timeSinceLastCheck() time.Duration {
	config := loadConfig()
	cached, ok := config[lastCheckCacheKey()]
	if !ok {
		return StatusCheckInterval + time.Hour // 超过阈值，需要立即校验
	}
	lastCheck, ok := cached.(float64)
	if !ok {
		return StatusCheckInterval + time.Hour
	}
	elapsed := time.Since(time.Unix(int64(lastCheck), 0))
	return elapsed
}

// 检查本地审批状态
func isLocallyApproved() bool {
	config := loadConfig()
	cached, ok := config[approvalCacheKey()].(map[string]interface{})
	if !ok {
		return false
	}
	approved, _ := cached["approved"].(bool)
	return approved
}

// 保存本地审批状态
func saveLocalApproval(record map[string]interface{}) error {
	config := loadConfig()
	config[approvalCacheKey()] = map[string]interface{}{
		"approved":   true,
		"deviceId":   deviceID(),
		"approvedAt": time.Now().Unix(),
		"record":     record,
	}
	return saveConfig(config)
}

// 清除本地审批状态
func clearLocalApproval() error {
	config := loadConfig()
	delete(config, approvalCacheKey())
	return saveConfig(config)
}

// 标记请求已提交
func markRequestSubmitted() error {
	config := loadConfig()
	config[requestCacheKey()] = map[string]interface{}{
		"submittedAt": time.Now().Unix(),
		"deviceId":    deviceID(),
	}
	return saveConfig(config)
}

// 距离下次请求的秒数
func secondsUntilNextRequest() int {
	config := loadConfig()
	cached, ok := config[requestCacheKey()].(map[string]interface{})
	if !ok {
		return 0
	}
	submittedAt, ok := cached["submittedAt"].(float64)
	if !ok {
		return 0
	}
	elapsed := int(time.Now().Unix() - int64(submittedAt))
	remaining := RequestIntervalSeconds - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// 格式化时长
func formatDuration(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	}
	if minutes < 1 {
		return "1分钟"
	}
	return fmt.Sprintf("%d分钟", minutes)
}

// API 响应结构
type APIResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

// API 通用响应
type APIStatusResponse struct {
	Ok          bool                   `json:"ok"`
	Approved    bool                   `json:"approved"`
	Blacklisted bool                   `json:"blacklisted"`
	Record      map[string]interface{} `json:"record"`
}

// 状态响应
type StatusResponse struct {
	Approved    bool
	Blacklisted bool
	Record      map[string]interface{}
}

// newHTTPClient 创建禁用 keep-alive 的 HTTP 客户端，避免 Cloudflare 复用连接导致的 EOF/握手错误
func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DisableKeepAlives:   true,
			MaxIdleConns:        0,
			MaxIdleConnsPerHost: 0,
			IdleConnTimeout:     10 * time.Second,
			ForceAttemptHTTP2:   false,
		},
	}
}

// doWithRetry 执行请求并对网络类错误重试，缓解 Cloudflare 偶发 TLS 握手/连接重置
func doWithRetry(method, url string, body string, timeout time.Duration) (*http.Response, []byte, error) {
	var lastErr error
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client := newHTTPClient(timeout)
		var reqBody io.Reader
		if body != "" {
			reqBody = strings.NewReader(body)
		}
		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			return nil, nil, err
		}
		addDefaultHeaders(req)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			// 仅对网络类错误重试
			if attempt < maxAttempts && isNetworkError(err) {
				time.Sleep(time.Duration(attempt) * 600 * time.Millisecond)
				continue
			}
			return nil, nil, err
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < maxAttempts {
				time.Sleep(time.Duration(attempt) * 600 * time.Millisecond)
				continue
			}
			return nil, nil, readErr
		}
		return resp, data, nil
	}
	return nil, nil, lastErr
}

// 带主备切换的 API 请求（优先主 API，失败后切换到兜底 API）
func doAPIRequest(method, path string, body string, timeout time.Duration) (*http.Response, []byte, error) {
	// 先尝试主 API
	resp, data, err := doWithRetry(method, APIBase+path, body, timeout)
	if err == nil {
		return resp, data, nil
	}

	// 主 API 失败，尝试兜底 API
	return doWithRetry(method, APIBaseFallback+path, body, timeout)
}

// 检查 API 可达性
func checkAPIReachable(timeout time.Duration) error {
	resp, _, err := doAPIRequest("GET", "/health", "", timeout)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API 返回状态码: %d", resp.StatusCode)
	}
	return nil
}

// 高频探测真正的公共互联网是否可达（10s一次）
func checkPublicInternet() error {
	// 使用国内极度稳定的公共 DNS 进行 TCP 三次握手探测，100% 精准判定公网连通性
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", "119.29.29.29:53") // 腾讯公共 DNS
	if err != nil {
		// 腾讯不通时，尝试阿里公共 DNS 双保险
		conn, err = d.Dial("tcp", "223.5.5.5:53")
		if err != nil {
			return err // 两个国内顶级大厂都连不上，说明真的彻底断网了
		}
	}
	conn.Close()
	return nil
}

// 检查 sdp.exe 进程是否在运行
func isSDPProcessRunning() bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq "+SDPProcessName, "/NH")
	hideWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(output)), strings.ToLower(SDPProcessName))
}

// 应用层真实网络校验：访问微软连接测试 URL
func checkApplicationLayerNetwork() error {
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	resp, err := client.Get(MSFTConnectTestURL)
	if err != nil {
		return fmt.Errorf("网络请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	if !strings.Contains(string(body), MSFTConnectTestKeyword) {
		return fmt.Errorf("响应内容不符合预期")
	}

	return nil
}

// 写入守护进程日志
func writeGuardLog(format string, args ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			// 日志写入失败不中断程序
		}
	}()

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(f, "[%s] %s\n", timestamp, message)
}

// 添加默认请求头（简化版，避免与 Cloudflare Worker 冲突）
func addDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Fuck0TrustClient/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
}

// 从 API 刷新审批状态
func refreshApprovalFromAPI(timeout time.Duration) (*StatusResponse, error) {
	path := fmt.Sprintf("/api/status?deviceId=%s", deviceID())

	resp, body, err := doAPIRequest("GET", path, "", timeout)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误: %s", string(body))
	}

	var apiResp APIStatusResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	result := &StatusResponse{
		Approved:    apiResp.Approved,
		Blacklisted: apiResp.Blacklisted,
		Record:      apiResp.Record,
	}
	
	// 更新本地缓存：被拉黑或未通过都清除本地审批
	if result.Approved && !result.Blacklisted {
		saveLocalApproval(result.Record)
	} else {
		clearLocalApproval()
	}
	
	return result, nil
}

// 提交审批请求
func requestApproval(note string) error {
	note = strings.TrimSpace(note)
	if note == "" {
		return fmt.Errorf("请填写你的可联系方式或申请理由，否则不予通过")
	}
	remaining := secondsUntilNextRequest()
	if remaining > 0 {
		return fmt.Errorf("同一设备 24 小时内只允许提交一次审批，请 %s 后再试", formatDuration(remaining))
	}
	
	hostname, _ := os.Hostname()
	currentUser, _ := user.Current()
	username := "unknown"
	if currentUser != nil {
		username = currentUser.Username
	}
	
	payload := map[string]interface{}{
		"deviceId": deviceID(),
		"hostname": hostname,
		"username": username,
		"note":     note,
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 使用带主备切换的 API 请求
	resp, body, err := doAPIRequest("POST", "/api/request", string(jsonData), DefaultConnectTimeout+DefaultReadTimeout)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("提交失败: %s", string(body))
	}

	markRequestSubmitted()
	return nil
}

// 检查是否是管理员
func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

// 自动定位并寻找 SDP 目录路径
func findSDPPath() (string, error) {
	subPath := filepath.Join("SDP", "ztgClient", "AccInject")
	
	// 遍历 Windows 所有可能的盘符
	for c := 'C'; c <= 'Z'; c++ {
		drive := fmt.Sprintf("%c:\\", c)
		targetPath := filepath.Join(drive, subPath)
		
		// 检查该目录是否存在
		if fi, err := os.Stat(targetPath); err == nil && fi.IsDir() {
			return targetPath, nil
		}
	}
	
	// 如果没找到，退回默认的 D 盘路径
	return "D:\\SDP\\ztgClient\\AccInject", fmt.Errorf("未在任意盘符中定位到 SDP 安装目录")
}

// 查询 WFP 服务状态 (根据新逻辑，此项改为定位并检查 ztgLoader 的存在性)
func queryWFPStatus() error {
	sdpPath, err := findSDPPath()
	if err != nil {
		fmt.Printf("[WARN] 自动定位失败，使用默认路径。错误: %v\n", err)
	}
	fmt.Printf("[INFO] 定位到目标路径：%s\n", sdpPath)
	return nil
}

// 停止 WFP 服务部分替换为调用 ztgLoader 卸载驱动
func stopWFPService() error {
	// 打印并定位路径
	_ = queryWFPStatus()

	sdpPath, _ := findSDPPath()
	loaderExe := filepath.Join(sdpPath, "ztgLoader.exe")

	fmt.Printf("[INFO] 正在切换至路径并执行卸载: %s\n", sdpPath)
	
	// 在 Go 中，设置 Cmd.Dir 相当于在执行前进行 cd /d
	cmd := exec.Command(loaderExe, "-u", "AccInject10_x64.sys")
	cmd.Dir = sdpPath
	
	hideWindow(cmd)

	output, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", string(output))
	if err != nil {
		return fmt.Errorf("执行 ztgLoader 失败: %v", err)
	}
	fmt.Printf("[INFO] 驱动卸载指令执行完毕。\n")
	return nil
}

// 执行一次受控功能
func runOnce() error {
	if !isLocallyApproved() {
		return fmt.Errorf("当前设备未审批通过，不能执行受控功能。请先打开客户端联网完成审批状态同步。")
	}
	return stopWFPService()
}

// 守护进程主循环（NetCheck.bat 逻辑的 Go 实现）
func guardLoop() {
	writeGuardLog("守护进程启动")

	// 创建互斥锁防止重复启动
	mutex, err := createMutex(GuardMutexName)
	if err != nil {
		writeGuardLog("守护进程已在运行，退出")
		return
	}
	defer releaseMutex(mutex)

	count := 0
	lastCheckTime := time.Now()

	// 启动时检查一次审批状态
	if timeSinceLastCheck() >= StatusCheckInterval {
		if !checkAndHandleApprovalStatus() {
			return // 状态异常，退出
		}
		saveLastCheckTime()
		lastCheckTime = time.Now()
	}

	for {
		// 定期校验审批状态（每 24 小时）
		if time.Since(lastCheckTime) >= StatusCheckInterval {
			writeGuardLog("执行 24 小时定期审批状态校验")
			if !checkAndHandleApprovalStatus() {
				writeGuardLog("审批状态异常，守护进程退出")
				return // 状态异常，自动停止
			}
			saveLastCheckTime()
			lastCheckTime = time.Now()
		}

		// 步骤 1：检查 sdp.exe 进程
		if !isSDPProcessRunning() {
			time.Sleep(ProcessCheckWaitTime)
			continue
		}

		// 步骤 2：应用层真实网络校验
		err := checkApplicationLayerNetwork()

		// 步骤 3：断网触发修复逻辑
		if err != nil {
			count++
			writeGuardLog("真实断网，执行修复程序，累计次数：%d", count)

			// 执行修复功能（调用 ztgLoader 卸载驱动）
			if errFix := stopWFPService(); errFix != nil {
				writeGuardLog("修复执行失败: %v", errFix)
			} else {
				writeGuardLog("修复执行成功")
			}

			// 修复后冷却
			time.Sleep(FixCooldownTime)
		}

		// 正常状态：休眠等待下一轮
		time.Sleep(GuardLoopInterval)
	}
}

// 检查并处理审批状态（状态异常时自动清理）
func checkAndHandleApprovalStatus() bool {
	status, err := refreshApprovalFromAPI(20 * time.Second)
	if err != nil {
		writeGuardLog("联网校验失败: %v", err)
		return true // 网络错误时不停止守护，继续运行
	}

	// 状态异常：被拉黑或未通过
	if status.Blacklisted || !status.Approved {
		writeGuardLog("审批状态异常 - 拉黑: %v, 通过: %v", status.Blacklisted, status.Approved)
		writeGuardLog("开始清理：删除计划任务...")

		// 尝试删除计划任务（可能失败但不影响退出）
		if isAdmin() {
			cmd := exec.Command("schtasks", "/Delete", "/TN", TaskName, "/F")
			hideWindow(cmd)
			if err := cmd.Run(); err != nil {
				writeGuardLog("删除计划任务失败: %v", err)
			} else {
				writeGuardLog("计划任务已删除")
			}
		} else {
			writeGuardLog("无管理员权限，跳过删除计划任务")
		}

		return false // 返回 false 表示需要停止守护
	}

	writeGuardLog("审批状态正常，继续守护")
	return true
}

// 创建 Windows 命名互斥锁
func createMutex(name string) (windows.Handle, error) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}

	handle, err := windows.CreateMutex(nil, true, namePtr)
	if err != nil {
		// 检查互斥锁是否已存在
		if err == windows.ERROR_ALREADY_EXISTS {
			if handle != 0 {
				windows.CloseHandle(handle)
			}
			return 0, fmt.Errorf("守护进程已在运行")
		}
		return 0, err
	}

	return handle, nil
}

// 释放互斥锁
func releaseMutex(handle windows.Handle) {
	if handle != 0 {
		windows.ReleaseMutex(handle)
		windows.CloseHandle(handle)
	}
}

// 获取当前可执行文件路径
func currentExePath() (string, error) {
	return os.Executable()
}

// 安装计划任务
func installTask() error {
	if !isLocallyApproved() {
		return fmt.Errorf("当前设备未审批通过，不能安装计划任务")
	}

	if !isAdmin() {
		return fmt.Errorf("写入系统计划任务需要管理员权限，请右键以管理员身份运行")
	}

	exePath, err := currentExePath()
	if err != nil {
		return err
	}

	// 1. 动态获取当前运行程序的管理员用户名
	currentUser, err := user.Current()
	username := "Administrators" // 备用降级值
	if err == nil && currentUser.Username != "" {
		username = currentUser.Username
	}

	// 2. 获取当前 software 所在的文件夹绝对路径
	exeDir := filepath.Dir(exePath)

	// 3. 计划任务改为"系统登录时在后台启动守护进程（NetCheck 模式）"
	cmd := exec.Command("schtasks",
		"/Create",
		"/TN", TaskName,
		"/TR", fmt.Sprintf(`"%s" guard`, exePath),  // 使用 guard 命令启动守护模式
		"/SC", "ONLOGON",          // 开机登录时自动在后台默默运行
		"/RL", "HIGHEST",          // 保持最高权限
		"/RU", username,           // 使用动态获取到的管理员账户
		"/F",
	)

	// 4. 确保执行时以当前程序所在文件夹作为起点
	cmd.Dir = exeDir
	hideWindow(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("创建计划任务失败: %s", string(output))
	}

	fmt.Printf("计划任务已创建/更新：%s，已开启开机自动守护（NetCheck 模式）。\n", TaskName)

	// 立即启动独立的后台守护进程（不依赖当前进程）
	startCmd := exec.Command(exePath, "guard")
	startCmd.Dir = exeDir
	startCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000 | 0x00000008, // CREATE_NO_WINDOW | DETACHED_PROCESS
	}

	if err := startCmd.Start(); err != nil {
		writeGuardLog("启动独立守护进程失败: %v", err)
		return fmt.Errorf("计划任务已安装，但启动守护进程失败: %v", err)
	}

	writeGuardLog("计划任务安装完成，独立守护进程已启动 PID: %d", startCmd.Process.Pid)
	fmt.Printf("守护进程已在后台启动 (PID: %d)\n", startCmd.Process.Pid)

	return nil
}

// 删除计划任务并停止所有守护进程
func removeTask() error {
	if !isAdmin() {
		return fmt.Errorf("删除系统计划任务需要管理员权限，请右键以管理员身份运行")
	}

	// 1. 先停止所有守护进程
	if err := stopAllGuardProcesses(); err != nil {
		fmt.Printf("警告：停止守护进程时出错: %v\n", err)
	}

	// 2. 删除计划任务
	cmd := exec.Command("schtasks", "/Delete", "/TN", TaskName, "/F")
	hideWindow(cmd)
	cmd.Run()
	fmt.Printf("计划任务已删除：%s\n", TaskName)

	return nil
}

// 仅停止守护进程，不删除计划任务
func stopGuard() error {
	return stopAllGuardProcesses()
}

// 停止所有守护进程
func stopAllGuardProcesses() error {
	exePath, err := currentExePath()
	if err != nil {
		return err
	}
	exeName := filepath.Base(exePath)

	// 使用 taskkill 强制结束所有同名进程（除了当前进程）
	cmd := exec.Command("taskkill", "/F", "/IM", exeName)
	hideWindow(cmd)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// taskkill 找不到进程会返回错误，但这不是真正的错误
		if strings.Contains(string(output), "not found") || strings.Contains(string(output), "找不到") {
			fmt.Println("没有运行中的守护进程")
			return nil
		}
		return fmt.Errorf("停止守护进程失败: %s", string(output))
	}

	fmt.Println("所有守护进程已停止")
	writeGuardLog("守护进程已被手动停止")
	return nil
}

// 隐藏 Windows 命令窗口
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

// 友好的网络错误提示
func friendlyNetworkError() string {
	return "网络连接失败：当前网络无法稳定访问审批服务，请稍后重试，或更换网络/检查代理后重新打开客户端。"
}

// apiHost 返回后端主机名（不含协议），用于错误脱敏
func apiHost() string {
	host := strings.TrimPrefix(APIBase, "https://")
	host = strings.TrimPrefix(host, "http://")
	return host
}

// isNetworkError 判断是否为网络类错误（关键词尽量覆盖全面）
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	keywords := []string{
		"timeout", "connection", "network", "dial", "eof",
		"no such host", "lookup", "refused", "unreachable",
		"tls", "handshake", "reset", "i/o timeout",
		"context deadline", "deadline exceeded", "wsarecv", "wsasend",
		"actively refused", "forcibly closed",
	}
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	// 错误信息中包含后端地址时，一律视为网络错误并脱敏
	if strings.Contains(s, strings.ToLower(apiHost())) {
		return true
	}
	return false
}

// sanitizeError 生成对用户安全的错误信息，绝不泄漏后端 API 地址
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	if isNetworkError(err) {
		return friendlyNetworkError()
	}
	// 兜底：即便不是网络错误，也把可能出现的后端地址替换掉
	msg := err.Error()
	msg = strings.ReplaceAll(msg, APIBase, "审批服务")
	msg = strings.ReplaceAll(msg, apiHost(), "审批服务")
	return msg
}

func main() {
	// 启动诊断日志
	startupLog := filepath.Join(os.TempDir(), "fuck0trust_startup.log")
	logStartup := func(msg string) {
		if f, err := os.OpenFile(startupLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
			f.Close()
		}
	}

	logStartup("=== 程序启动 ===")
	logStartup(fmt.Sprintf("参数数量: %d", len(os.Args)))
	if len(os.Args) > 0 {
		logStartup(fmt.Sprintf("参数列表: %v", os.Args))
	}

	// 捕获 panic 并记录到文件，支持输出完整的高精度红字代码行数堆栈
	defer func() {
		if r := recover(); r != nil {
			logFile := filepath.Join(os.TempDir(), "fuck0trust_crash.log")
			f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				fmt.Fprintf(f, "\n=== Crash at %s ===\n", time.Now().Format("2006-01-02 15:04:05"))
				fmt.Fprintf(f, "Panic: %v\n", r)
				fmt.Fprintf(f, "Device ID: %s\n", deviceID())
				f.Write(debug.Stack()) // 👈 【核心修复】直接暴露精准崩溃函数位置，不留任何死角
				f.Close()
			}
			// 生产环境不重新抛出 panic，避免闪退
			fmt.Fprintf(os.Stderr, "程序异常退出: %v\n", r)
			fmt.Fprintf(os.Stderr, "详细日志已保存到: %s\n", logFile)
			logStartup(fmt.Sprintf("PANIC: %v", r))
			os.Exit(1)
		}
	}()

	if len(os.Args) == 1 {
		// 无参数,启动 GUI
		logStartup("准备启动 GUI 模式")
		launchGUI()
		logStartup("GUI 模式已退出")
		return
	}
	
	// 命令行模式
	if len(os.Args) < 2 {
		fmt.Println(`用法: Fuck0TrustClient.exe [命令]`)
		fmt.Println(`命令:`)
		fmt.Println(`  request [--note 备注]  - 提交审批申请`)
		fmt.Println(`  status                  - 查询审批状态`)
		fmt.Println(`  guard                   - 启动守护进程（NetCheck 模式）`)
		fmt.Println(`  install-task            - 安装计划任务`)
		fmt.Println(`  remove-task             - 删除计划任务并停止守护进程`)
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "request":
		note := ""
		if len(os.Args) > 3 && os.Args[2] == "--note" {
			note = os.Args[3]
		}
		if err := requestApproval(note); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\n设备 ID: %s\n", deviceID())
		fmt.Println("已提交审批申请，请联系管理员审批。")
		
	case "status":
		did := deviceID()
		status, err := refreshApprovalFromAPI(20 * time.Second)
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		data, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(data))
		fmt.Printf("\n设备 ID: %s\n", did)
		if status.Blacklisted {
			fmt.Println("审批状态：已被拉黑，请联系 @pppatr1ck_bot")
			os.Exit(0)
		} else if status.Approved {
			fmt.Println("审批状态：已通过")
		} else {
			fmt.Println("审批状态：未通过/待审批")
		}

	case "guard":
		// 守护模式：NetCheck.bat 的完整逻辑
		if !isLocallyApproved() {
			fmt.Fprintf(os.Stderr, "错误: 当前设备未审批通过，不能启动守护进程\n")
			os.Exit(1)
		}
		fmt.Println("[INFO] 启动守护进程（NetCheck 模式）...")
		guardLoop()

	case "install-task":
		if err := installTask(); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

	case "remove-task":
		if err := removeTask(); err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n", command)
		os.Exit(1)
	}
}