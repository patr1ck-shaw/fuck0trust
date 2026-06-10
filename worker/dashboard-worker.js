const jsonHeaders = {
  "content-type": "application/json; charset=utf-8",
  "access-control-allow-origin": "*",
  "access-control-allow-methods": "GET,POST,OPTIONS",
  "access-control-allow-headers": "content-type,authorization"
};

function json(data, init = {}) {
  return new Response(JSON.stringify(data, null, 2), {
    status: init.status || 200,
    headers: Object.assign({}, jsonHeaders, init.headers || {})
  });
}

function html(body) {
  return new Response(body, { headers: { "content-type": "text/html; charset=utf-8" } });
}

function redirect(location, headers = {}) {
  return new Response(null, { status: 302, headers: Object.assign({ Location: location }, headers) });
}

function esc(v) {
  return String(v == null ? "" : v).replace(/[&<>"']/g, function (c) {
    return { "&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;", "'": "&#39;" }[c];
  });
}

function deviceKey(deviceId) {
  return "device:" + deviceId;
}

function validateDeviceId(deviceId) {
  return typeof deviceId === "string" && /^[a-f0-9]{64}$/i.test(deviceId);
}

function cookies(request) {
  const out = {};
  (request.headers.get("cookie") || "").split(";").forEach(function (p) {
    const i = p.indexOf("=");
    if (i > -1) out[p.slice(0, i).trim()] = decodeURIComponent(p.slice(i + 1).trim());
  });
  return out;
}

function loggedIn(request) {
  return ADMIN_TOKEN && ADMIN_TOKEN !== "CHANGE_ME_ADMIN_TOKEN" && cookies(request).admin_token === ADMIN_TOKEN;
}

function requireAdmin(request) {
  const auth = request.headers.get("authorization") || "";
  if (!ADMIN_TOKEN || ADMIN_TOKEN === "CHANGE_ME_ADMIN_TOKEN" || auth !== "Bearer " + ADMIN_TOKEN) {
    return json({ ok: false, error: "未授权的管理员请求" }, { status: 401 });
  }
  return null;
}

async function readBody(request) {
  try {
    return await request.json();
  } catch (e) {
    throw new Error("请求体必须是 JSON");
  }
}

async function listRecords() {
  const list = await DEVICE_APPROVAL_KV.list({ prefix: "device:" });
  const records = await Promise.all(list.keys.map(function (k) {
    return DEVICE_APPROVAL_KV.get(k.name, "json");
  }));
  return records.filter(Boolean).sort(function (a, b) {
    return String(b.updatedAt || "").localeCompare(String(a.updatedAt || ""));
  });
}

function shell(content) {
  return html('<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>设备审批管理</title><style>body{font-family:Arial,"Microsoft YaHei",sans-serif;background:#f6f7fb;margin:0;color:#172033}.wrap{max-width:1180px;margin:auto;padding:26px 16px}.card{background:#fff;border:1px solid #e5e7eb;border-radius:12px;padding:18px;margin-bottom:16px;box-shadow:0 8px 24px #0001}input{padding:10px;border:1px solid #cbd5e1;border-radius:8px;min-width:280px}button{border:0;border-radius:8px;padding:9px 13px;background:#2563eb;color:white;font-weight:700;cursor:pointer}.secondary{background:#64748b}.approve{background:#16a34a}.deny{background:#dc2626}table{width:100%;border-collapse:collapse}th,td{padding:10px;border-bottom:1px solid #e5e7eb;text-align:left;vertical-align:top}th{background:#f8fafc}code{word-break:break-all;background:#f1f5f9;padding:2px 5px;border-radius:5px}.pending{color:#92400e}.approved{color:#166534}.denied{color:#991b1b}.msg{padding:10px;border-radius:8px;background:#e0f2fe;margin:10px 0}.err{background:#fee2e2}.actions{display:flex;gap:8px;flex-wrap:wrap}.inline{display:inline}</style></head><body><div class="wrap">' + content + '</div></body></html>');
}

function loginPage(message, err) {
  return shell('<div class="card"><h1>设备审批管理面板</h1><p>输入 Worker 里设置的 <code>ADMIN_TOKEN</code> 登录。</p>' + (message ? '<div class="msg ' + (err ? 'err' : '') + '">' + esc(message) + '</div>' : '') + '<form method="post" action="/admin/login"><input name="token" type="password" placeholder="ADMIN_TOKEN" required> <button>登录</button></form></div>');
}

function adminPage(records, message, err) {
  const rows = records.length ? records.map(function (r) {
    const id = esc(r.deviceId);
    return '<tr><td class="' + esc(r.status) + '"><b>' + esc(r.status) + '</b></td><td><code>' + id + '</code></td><td>' + esc(r.hostname || '-') + '<br>' + esc(r.username || '-') + '</td><td>' + esc(r.note || '-') + '</td><td>' + esc(r.updatedAt || r.requestedAt || '-') + '</td><td><div class="actions"><form class="inline" method="post" action="/admin/decision"><input type="hidden" name="deviceId" value="' + id + '"><input type="hidden" name="action" value="approve"><button class="approve">批准</button></form><form class="inline" method="post" action="/admin/decision"><input type="hidden" name="deviceId" value="' + id + '"><input type="hidden" name="action" value="deny"><button class="deny">拒绝</button></form></div></td></tr>';
  }).join("") : '<tr><td colspan="6">暂无设备申请。客户端提交后会显示在这里。</td></tr>';
  return shell('<div class="card"><h1>设备审批管理面板</h1><p>已登录。这个版本不靠前端 JS 刷新，页面打开时由 Worker 直接读取 KV。</p><div class="actions"><form method="get" action="/admin"><button class="secondary">刷新设备</button></form><form method="post" action="/admin/logout"><button class="secondary">退出登录</button></form></div>' + (message ? '<div class="msg ' + (err ? 'err' : '') + '">' + esc(message) + '</div>' : '') + '</div><div class="card"><h2>设备列表：' + records.length + ' 台</h2><div style="overflow:auto"><table><thead><tr><th>状态</th><th>设备ID</th><th>主机/用户</th><th>备注</th><th>更新时间</th><th>操作</th></tr></thead><tbody>' + rows + '</tbody></table></div></div>');
}

async function setStatus(deviceId, status) {
  const key = deviceKey(deviceId);
  const old = await DEVICE_APPROVAL_KV.get(key, "json");
  if (!old) return { ok: false, error: "设备申请不存在" };
  const now = new Date().toISOString();
  const record = Object.assign({}, old, { status: status, updatedAt: now, approvedAt: status === "approved" ? now : old.approvedAt, deniedAt: status === "denied" ? now : old.deniedAt });
  await DEVICE_APPROVAL_KV.put(key, JSON.stringify(record));
  return { ok: true, record: record };
}

async function handleAdmin(request, url) {
  if (request.method === "GET") {
    if (!loggedIn(request)) return loginPage();
    return adminPage(await listRecords(), url.searchParams.get("msg") || url.searchParams.get("error") || "", !!url.searchParams.get("error"));
  }
  return json({ ok: false, error: "Method Not Allowed" }, { status: 405 });
}

async function handleLogin(request) {
  const form = await request.formData();
  const token = String(form.get("token") || "");
  if (!ADMIN_TOKEN || ADMIN_TOKEN === "CHANGE_ME_ADMIN_TOKEN") return loginPage("请先在 Worker 变量/Secret 中设置 ADMIN_TOKEN。", true);
  if (token !== ADMIN_TOKEN) return loginPage("ADMIN_TOKEN 不正确。", true);
  return redirect("/admin", { "Set-Cookie": "admin_token=" + encodeURIComponent(token) + "; Path=/admin; HttpOnly; Secure; SameSite=Lax; Max-Age=86400" });
}

async function handleDecision(request) {
  if (!loggedIn(request)) return redirect("/admin");
  const form = await request.formData();
  const id = String(form.get("deviceId") || "");
  const action = String(form.get("action") || "");
  const status = action === "approve" ? "approved" : action === "deny" ? "denied" : "";
  if (!validateDeviceId(id) || !status) return redirect("/admin?error=" + encodeURIComponent("参数错误"));
  const r = await setStatus(id, status);
  return redirect("/admin?" + (r.ok ? "msg=" + encodeURIComponent(status === "approved" ? "已批准" : "已拒绝") : "error=" + encodeURIComponent(r.error)));
}

async function handleRequestApproval(request) {
  const body = await readBody(request);
  if (!validateDeviceId(body.deviceId)) return json({ ok: false, error: "deviceId 必须是 64 位十六进制 SHA-256" }, { status: 400 });
  const now = new Date().toISOString();
  const key = deviceKey(body.deviceId);
  const old = await DEVICE_APPROVAL_KV.get(key, "json");
  const record = old ? Object.assign({}, old, { hostname: body.hostname || old.hostname, username: body.username || old.username, note: body.note || old.note, updatedAt: now }) : { deviceId: body.deviceId, status: "pending", hostname: body.hostname, username: body.username, note: body.note, requestedAt: now, updatedAt: now };
  await DEVICE_APPROVAL_KV.put(key, JSON.stringify(record));
  return json({ ok: true, record: record });
}

async function handleStatus(url) {
  const id = url.searchParams.get("deviceId");
  if (!validateDeviceId(id)) return json({ ok: false, error: "缺少合法 deviceId" }, { status: 400 });
  const record = await DEVICE_APPROVAL_KV.get(deviceKey(id), "json");
  return json({ ok: true, approved: !!record && record.status === "approved", record: record || null });
}

async function fetchHandler(request) {
  if (request.method === "OPTIONS") return new Response(null, { headers: jsonHeaders });
  const url = new URL(request.url);
  try {
    if (url.pathname === "/admin") return await handleAdmin(request, url);
    if (request.method === "POST" && url.pathname === "/admin/login") return await handleLogin(request);
    if (request.method === "POST" && url.pathname === "/admin/logout") return redirect("/admin", { "Set-Cookie": "admin_token=; Path=/admin; HttpOnly; Secure; SameSite=Lax; Max-Age=0" });
    if (request.method === "POST" && url.pathname === "/admin/decision") return await handleDecision(request);
    if (request.method === "GET" && (url.pathname === "/" || url.pathname === "/health")) return json({ ok: true, service: "device-approval-worker" });
    if (request.method === "POST" && url.pathname === "/api/request") return await handleRequestApproval(request);
    if (request.method === "GET" && url.pathname === "/api/status") return await handleStatus(url);
    if (request.method === "GET" && url.pathname === "/api/admin/devices") { const e = requireAdmin(request); return e || json({ ok: true, records: await listRecords() }); }
    if (request.method === "POST" && url.pathname === "/api/admin/approve") { const e = requireAdmin(request); if (e) return e; const b = await readBody(request); return json(await setStatus(b.deviceId, "approved")); }
    if (request.method === "POST" && url.pathname === "/api/admin/deny") { const e = requireAdmin(request); if (e) return e; const b = await readBody(request); return json(await setStatus(b.deviceId, "denied")); }
    return json({ ok: false, error: "Not Found" }, { status: 404 });
  } catch (error) {
    return json({ ok: false, error: error instanceof Error ? error.message : String(error) }, { status: 500 });
  }
}

addEventListener("fetch", function (event) {
  event.respondWith(fetchHandler(event.request));
});