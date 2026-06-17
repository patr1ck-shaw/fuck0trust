package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/dgraph-io/badger/v4"
)

var (
	db         *badger.DB
	adminToken string
)

type DeviceRecord struct {
	DeviceID      string `json:"deviceId"`
	Status        string `json:"status"`
	Hostname      string `json:"hostname,omitempty"`
	Username      string `json:"username,omitempty"`
	Note          string `json:"note,omitempty"`
	RequestedAt   string `json:"requestedAt,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
	ApprovedAt    string `json:"approvedAt,omitempty"`
	DeniedAt      string `json:"deniedAt,omitempty"`
	BlacklistedAt string `json:"blacklistedAt,omitempty"`
}

func main() {
	adminToken = os.Getenv("ADMIN_TOKEN")
	if adminToken == "" || adminToken == "CHANGE_ME_ADMIN_TOKEN" {
		log.Println("警告: ADMIN_TOKEN 未设置或使用默认值，请设置环境变量 ADMIN_TOKEN")
	}

	// 初始化 BadgerDB
	opts := badger.DefaultOptions("./data/badger")
	opts.Logger = nil                    // 禁用 BadgerDB 日志
	opts.ValueLogFileSize = 64 << 20     // 64MB (默认 2GB，对小规模数据太大)
	opts.ValueLogMaxEntries = 100000     // 限制单文件条目数
	var err error
	db, err = badger.Open(opts)
	if err != nil {
		log.Fatal("无法打开 BadgerDB:", err)
	}
	defer db.Close()

	// 路由
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/admin", handleAdmin)
	http.HandleFunc("/admin/login", handleLogin)
	http.HandleFunc("/admin/logout", handleLogout)
	http.HandleFunc("/admin/decision", handleDecision)
	http.HandleFunc("/api/request", handleRequestApproval)
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/admin/devices", handleAdminDevices)
	http.HandleFunc("/api/admin/approve", handleAdminApprove)
	http.HandleFunc("/api/admin/deny", handleAdminDeny)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("服务启动在端口 %s", port)
	log.Fatal(http.ListenAndServe(":"+port, corsMiddleware(http.DefaultServeMux)))
}

// CORS 中间件
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// 工具函数
func deviceKey(deviceID string) string {
	return "device:" + deviceID
}

func validateDeviceID(deviceID string) bool {
	if len(deviceID) != 64 {
		return false
	}
	for _, c := range deviceID {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func jsonResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func htmlResponse(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, body)
}

// BadgerDB 操作
func getRecord(deviceID string) (*DeviceRecord, error) {
	var record DeviceRecord
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(deviceKey(deviceID)))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &record)
		})
	})
	if err == badger.ErrKeyNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func putRecord(record *DeviceRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(deviceKey(record.DeviceID)), data)
	})
}

func deleteRecord(deviceID string) error {
	return db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(deviceKey(deviceID)))
	})
}

func listRecords() ([]*DeviceRecord, error) {
	var records []*DeviceRecord
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("device:")
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var record DeviceRecord
				if err := json.Unmarshal(val, &record); err != nil {
					return err
				}
				records = append(records, &record)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 按更新时间倒序排序
	sort.Slice(records, func(i, j int) bool {
		return records[i].UpdatedAt > records[j].UpdatedAt
	})

	return records, nil
}

// Cookie 和认证
func getCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func isLoggedIn(r *http.Request) bool {
	return adminToken != "" && adminToken != "CHANGE_ME_ADMIN_TOKEN" && getCookie(r, "admin_token") == adminToken
}

func requireAdmin(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	return adminToken != "" && adminToken != "CHANGE_ME_ADMIN_TOKEN" && auth == "Bearer "+adminToken
}

// HTTP 处理函数
func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	jsonResponse(w, map[string]interface{}{"ok": true, "service": "fuck0trust-go-kv-server"}, http.StatusOK)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{"ok": true, "service": "fuck0trust-go-kv-server"}, http.StatusOK)
}

func handleRequestApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "Method Not Allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		DeviceID string `json:"deviceId"`
		Hostname string `json:"hostname"`
		Username string `json:"username"`
		Note     string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "请求体必须是 JSON"}, http.StatusBadRequest)
		return
	}

	if !validateDeviceID(body.DeviceID) {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "deviceId 必须是 64 位十六进制 SHA-256"}, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	old, err := getRecord(body.DeviceID)
	if err != nil {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "数据库错误"}, http.StatusInternalServerError)
		return
	}

	// 已被拉黑的设备不允许再次提交申请
	if old != nil && old.Status == "blacklisted" {
		jsonResponse(w, map[string]interface{}{"ok": false, "blacklisted": true, "error": "该设备已被拉黑"}, http.StatusForbidden)
		return
	}

	var record DeviceRecord
	if old != nil {
		record = *old
		if body.Hostname != "" {
			record.Hostname = body.Hostname
		}
		if body.Username != "" {
			record.Username = body.Username
		}
		if body.Note != "" {
			record.Note = body.Note
		}
		record.UpdatedAt = now
	} else {
		record = DeviceRecord{
			DeviceID:    body.DeviceID,
			Status:      "pending",
			Hostname:    body.Hostname,
			Username:    body.Username,
			Note:        body.Note,
			RequestedAt: now,
			UpdatedAt:   now,
		}
	}

	if err := putRecord(&record); err != nil {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "保存失败"}, http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{"ok": true, "record": record}, http.StatusOK)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "Method Not Allowed"}, http.StatusMethodNotAllowed)
		return
	}

	deviceID := r.URL.Query().Get("deviceId")
	if !validateDeviceID(deviceID) {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "缺少合法 deviceId"}, http.StatusBadRequest)
		return
	}

	record, err := getRecord(deviceID)
	if err != nil {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "数据库错误"}, http.StatusInternalServerError)
		return
	}

	blacklisted := record != nil && record.Status == "blacklisted"
	approved := record != nil && record.Status == "approved"

	jsonResponse(w, map[string]interface{}{
		"ok":          true,
		"approved":    approved,
		"blacklisted": blacklisted,
		"record":      record,
	}, http.StatusOK)
}

func handleAdminDevices(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "未授权的管理员请求"}, http.StatusUnauthorized)
		return
	}

	records, err := listRecords()
	if err != nil {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "数据库错误"}, http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{"ok": true, "records": records}, http.StatusOK)
}

func handleAdminApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "Method Not Allowed"}, http.StatusMethodNotAllowed)
		return
	}

	if !requireAdmin(r) {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "未授权的管理员请求"}, http.StatusUnauthorized)
		return
	}

	var body struct {
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "请求体必须是 JSON"}, http.StatusBadRequest)
		return
	}

	result := setStatus(body.DeviceID, "approved")
	jsonResponse(w, result, http.StatusOK)
}

func handleAdminDeny(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "Method Not Allowed"}, http.StatusMethodNotAllowed)
		return
	}

	if !requireAdmin(r) {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "未授权的管理员请求"}, http.StatusUnauthorized)
		return
	}

	var body struct {
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "请求体必须是 JSON"}, http.StatusBadRequest)
		return
	}

	result := setStatus(body.DeviceID, "denied")
	jsonResponse(w, result, http.StatusOK)
}

func setStatus(deviceID string, status string) map[string]interface{} {
	record, err := getRecord(deviceID)
	if err != nil {
		return map[string]interface{}{"ok": false, "error": "数据库错误"}
	}
	if record == nil {
		return map[string]interface{}{"ok": false, "error": "设备申请不存在"}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	record.Status = status
	record.UpdatedAt = now

	if status == "approved" {
		record.ApprovedAt = now
	} else if status == "denied" {
		record.DeniedAt = now
	} else if status == "blacklisted" {
		record.BlacklistedAt = now
	}

	if err := putRecord(record); err != nil {
		return map[string]interface{}{"ok": false, "error": "保存失败"}
	}

	return map[string]interface{}{"ok": true, "record": record}
}

// Web 管理界面（复用 worker 的 HTML）
func shell(content string) string {
	return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>设备审批管理</title><style>body{font-family:Arial,"Microsoft YaHei",sans-serif;background:#f6f7fb;margin:0;color:#172033}.wrap{max-width:1180px;margin:auto;padding:26px 16px}.card{background:#fff;border:1px solid #e5e7eb;border-radius:12px;padding:18px;margin-bottom:16px;box-shadow:0 8px 24px #0001}input{padding:10px;border:1px solid #cbd5e1;border-radius:8px;min-width:280px}button{border:0;border-radius:8px;padding:9px 13px;background:#2563eb;color:white;font-weight:700;cursor:pointer}.secondary{background:#64748b}.approve{background:#16a34a}.deny{background:#dc2626}.blacklist{background:#111827}.delete{background:#9333ea}table{width:100%;border-collapse:collapse}th,td{padding:10px;border-bottom:1px solid #e5e7eb;text-align:left;vertical-align:top}th{background:#f8fafc}code{word-break:break-all;background:#f1f5f9;padding:2px 5px;border-radius:5px}.pending{color:#92400e}.approved{color:#166534}.denied{color:#991b1b}.blacklisted{color:#111827;font-weight:700}.msg{padding:10px;border-radius:8px;background:#e0f2fe;margin:10px 0}.err{background:#fee2e2}.actions{display:flex;gap:8px;flex-wrap:wrap}.inline{display:inline}</style></head><body><div class="wrap">` + content + `</div></body></html>`
}

func loginPage(message string, isError bool) string {
	msgHTML := ""
	if message != "" {
		errClass := ""
		if isError {
			errClass = " err"
		}
		msgHTML = `<div class="msg` + errClass + `">` + html.EscapeString(message) + `</div>`
	}
	return shell(`<div class="card"><h1>设备审批管理面板</h1><p>输入服务器设置的 <code>ADMIN_TOKEN</code> 登录。</p>` + msgHTML + `<form method="post" action="/admin/login"><input name="token" type="password" placeholder="ADMIN_TOKEN" required> <button>登录</button></form></div>`)
}

func statusLabel(status string) string {
	switch status {
	case "approved":
		return "已通过"
	case "denied":
		return "已拒绝"
	case "blacklisted":
		return "已拉黑"
	default:
		return "待审批"
	}
}

func adminPage(records []*DeviceRecord, message string, isError bool) string {
	msgHTML := ""
	if message != "" {
		errClass := ""
		if isError {
			errClass = " err"
		}
		msgHTML = `<div class="msg` + errClass + `">` + html.EscapeString(message) + `</div>`
	}

	rows := ""
	if len(records) > 0 {
		for _, r := range records {
			id := html.EscapeString(r.DeviceID)
			hostname := html.EscapeString(r.Hostname)
			if hostname == "" {
				hostname = "-"
			}
			username := html.EscapeString(r.Username)
			if username == "" {
				username = "-"
			}
			note := html.EscapeString(r.Note)
			if note == "" {
				note = "-"
			}
			updatedAt := html.EscapeString(r.UpdatedAt)
			if updatedAt == "" {
				updatedAt = html.EscapeString(r.RequestedAt)
			}
			if updatedAt == "" {
				updatedAt = "-"
			}

			delConfirm := `return confirm('确定删除该设备申请记录吗？此操作不可恢复。')`
			blkConfirm := `return confirm('确定拉黑该设备吗？客户端同步状态时会被强制退出。')`

			act := func(action, cls, label, onsubmit string) string {
				submitAttr := ""
				if onsubmit != "" {
					submitAttr = ` onsubmit="` + onsubmit + `"`
				}
				return `<form class="inline" method="post" action="/admin/decision"` + submitAttr + `><input type="hidden" name="deviceId" value="` + id + `"><input type="hidden" name="action" value="` + action + `"><button class="` + cls + `">` + label + `</button></form>`
			}

			rows += `<tr><td class="` + r.Status + `"><b>` + html.EscapeString(statusLabel(r.Status)) + `</b></td><td><code>` + id + `</code></td><td>` + hostname + `<br>` + username + `</td><td>` + note + `</td><td>` + updatedAt + `</td><td><div class="actions">` + act("approve", "approve", "批准", "") + act("deny", "deny", "拒绝", "") + act("blacklist", "blacklist", "拉黑", blkConfirm) + act("delete", "delete", "删除", delConfirm) + `</div></td></tr>`
		}
	} else {
		rows = `<tr><td colspan="6">暂无设备申请。客户端提交后会显示在这里。</td></tr>`
	}

	return shell(`<div class="card"><h1>设备审批管理面板</h1><p>已登录。这个版本不靠前端 JS 刷新，页面打开时由服务器直接读取 KV。</p><div class="actions"><form method="get" action="/admin"><button class="secondary">刷新设备</button></form><form method="post" action="/admin/logout"><button class="secondary">退出登录</button></form></div>` + msgHTML + `</div><div class="card"><h2>设备列表：` + fmt.Sprintf("%d", len(records)) + ` 台</h2><div style="overflow:auto"><table><thead><tr><th>状态</th><th>设备ID</th><th>主机/用户</th><th>备注</th><th>更新时间</th><th>操作</th></tr></thead><tbody>` + rows + `</tbody></table></div></div>`)
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonResponse(w, map[string]interface{}{"ok": false, "error": "Method Not Allowed"}, http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		htmlResponse(w, loginPage("", false))
		return
	}

	records, err := listRecords()
	if err != nil {
		htmlResponse(w, adminPage([]*DeviceRecord{}, "数据库错误", true))
		return
	}

	message := r.URL.Query().Get("msg")
	errorMsg := r.URL.Query().Get("error")
	if errorMsg != "" {
		htmlResponse(w, adminPage(records, errorMsg, true))
		return
	}

	htmlResponse(w, adminPage(records, message, false))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		htmlResponse(w, loginPage("表单解析错误", true))
		return
	}

	token := r.FormValue("token")

	if adminToken == "" || adminToken == "CHANGE_ME_ADMIN_TOKEN" {
		htmlResponse(w, loginPage("请先设置环境变量 ADMIN_TOKEN。", true))
		return
	}

	if token != adminToken {
		htmlResponse(w, loginPage("ADMIN_TOKEN 不正确。", true))
		return
	}

	cookie := &http.Cookie{
		Name:     "admin_token",
		Value:    token,
		Path:     "/admin",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	cookie := &http.Cookie{
		Name:     "admin_token",
		Value:    "",
		Path:     "/admin",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func handleDecision(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin?error=表单解析错误", http.StatusSeeOther)
		return
	}

	deviceID := r.FormValue("deviceId")
	action := r.FormValue("action")

	if !validateDeviceID(deviceID) {
		http.Redirect(w, r, "/admin?error=参数错误", http.StatusSeeOther)
		return
	}

	if action == "delete" {
		if err := deleteRecord(deviceID); err != nil {
			http.Redirect(w, r, "/admin?error=删除失败", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/admin?msg=已删除该设备申请", http.StatusSeeOther)
		return
	}

	var status string
	var okMsg string
	switch action {
	case "approve":
		status = "approved"
		okMsg = "已批准"
	case "deny":
		status = "denied"
		okMsg = "已拒绝"
	case "blacklist":
		status = "blacklisted"
		okMsg = "已拉黑"
	default:
		http.Redirect(w, r, "/admin?error=参数错误", http.StatusSeeOther)
		return
	}

	result := setStatus(deviceID, status)
	if ok, _ := result["ok"].(bool); !ok {
		errorMsg := "操作失败"
		if err, exists := result["error"].(string); exists {
			errorMsg = err
		}
		http.Redirect(w, r, "/admin?error="+errorMsg, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin?msg="+okMsg, http.StatusSeeOther)
}

