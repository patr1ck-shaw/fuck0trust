export interface Env {
  DEVICE_APPROVAL_KV: KVNamespace;
  ADMIN_TOKEN: string;
}

type DeviceStatus = "pending" | "approved" | "denied";

interface DeviceRecord {
  deviceId: string;
  status: DeviceStatus;
  hostname?: string;
  username?: string;
  note?: string;
  requestedAt: string;
  updatedAt: string;
  approvedAt?: string;
  deniedAt?: string;
}

const jsonHeaders = {
  "content-type": "application/json; charset=utf-8",
  "access-control-allow-origin": "*",
  "access-control-allow-methods": "GET,POST,OPTIONS",
  "access-control-allow-headers": "content-type,authorization"
};

function json(data: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(data, null, 2), {
    ...init,
    headers: { ...jsonHeaders, ...(init.headers ?? {}) }
  });
}

function deviceKey(deviceId: string) {
  return `device:${deviceId}`;
}

function validateDeviceId(deviceId: unknown): deviceId is string {
  return typeof deviceId === "string" && /^[a-f0-9]{64}$/i.test(deviceId);
}

async function readBody<T>(request: Request): Promise<T> {
  try {
    return (await request.json()) as T;
  } catch {
    throw new Error("请求体必须是 JSON");
  }
}

function requireAdmin(request: Request, env: Env): Response | null {
  const auth = request.headers.get("authorization") ?? "";
  const expected = `Bearer ${env.ADMIN_TOKEN}`;
  if (!env.ADMIN_TOKEN || env.ADMIN_TOKEN === "CHANGE_ME_ADMIN_TOKEN" || auth !== expected) {
    return json({ ok: false, error: "未授权的管理员请求" }, { status: 401 });
  }
  return null;
}

async function handleRequestApproval(request: Request, env: Env) {
  const body = await readBody<{ deviceId?: string; hostname?: string; username?: string; note?: string }>(request);
  if (!validateDeviceId(body.deviceId)) {
    return json({ ok: false, error: "deviceId 必须是 64 位十六进制 SHA-256" }, { status: 400 });
  }

  const now = new Date().toISOString();
  const key = deviceKey(body.deviceId);
  const existing = await env.DEVICE_APPROVAL_KV.get<DeviceRecord>(key, "json");
  const record: DeviceRecord = existing
    ? {
        ...existing,
        hostname: body.hostname ?? existing.hostname,
        username: body.username ?? existing.username,
        note: body.note ?? existing.note,
        updatedAt: now
      }
    : {
        deviceId: body.deviceId,
        status: "pending",
        hostname: body.hostname,
        username: body.username,
        note: body.note,
        requestedAt: now,
        updatedAt: now
      };

  await env.DEVICE_APPROVAL_KV.put(key, JSON.stringify(record));
  return json({ ok: true, record });
}

async function handleStatus(url: URL, env: Env) {
  const deviceId = url.searchParams.get("deviceId");
  if (!validateDeviceId(deviceId)) {
    return json({ ok: false, error: "缺少合法 deviceId" }, { status: 400 });
  }
  const record = await env.DEVICE_APPROVAL_KV.get<DeviceRecord>(deviceKey(deviceId), "json");
  return json({ ok: true, approved: record?.status === "approved", record: record ?? null });
}

async function handleAdminList(request: Request, env: Env) {
  const authError = requireAdmin(request, env);
  if (authError) return authError;

  const list = await env.DEVICE_APPROVAL_KV.list({ prefix: "device:" });
  const records = await Promise.all(
    list.keys.map((key) => env.DEVICE_APPROVAL_KV.get<DeviceRecord>(key.name, "json"))
  );
  return json({ ok: true, records: records.filter(Boolean) });
}

async function handleAdminDecision(request: Request, env: Env, status: DeviceStatus) {
  const authError = requireAdmin(request, env);
  if (authError) return authError;

  const body = await readBody<{ deviceId?: string }>(request);
  if (!validateDeviceId(body.deviceId)) {
    return json({ ok: false, error: "缺少合法 deviceId" }, { status: 400 });
  }
  const key = deviceKey(body.deviceId);
  const existing = await env.DEVICE_APPROVAL_KV.get<DeviceRecord>(key, "json");
  if (!existing) {
    return json({ ok: false, error: "设备申请不存在" }, { status: 404 });
  }

  const now = new Date().toISOString();
  const record: DeviceRecord = {
    ...existing,
    status,
    updatedAt: now,
    approvedAt: status === "approved" ? now : existing.approvedAt,
    deniedAt: status === "denied" ? now : existing.deniedAt
  };
  await env.DEVICE_APPROVAL_KV.put(key, JSON.stringify(record));
  return json({ ok: true, record });
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    if (request.method === "OPTIONS") return new Response(null, { headers: jsonHeaders });

    const url = new URL(request.url);
    try {
      if (request.method === "GET" && url.pathname === "/health") {
        return json({ ok: true, service: "device-approval-worker" });
      }
      if (request.method === "POST" && url.pathname === "/api/request") {
        return await handleRequestApproval(request, env);
      }
      if (request.method === "GET" && url.pathname === "/api/status") {
        return await handleStatus(url, env);
      }
      if (request.method === "GET" && url.pathname === "/api/admin/devices") {
        return await handleAdminList(request, env);
      }
      if (request.method === "POST" && url.pathname === "/api/admin/approve") {
        return await handleAdminDecision(request, env, "approved");
      }
      if (request.method === "POST" && url.pathname === "/api/admin/deny") {
        return await handleAdminDecision(request, env, "denied");
      }
      return json({ ok: false, error: "Not Found" }, { status: 404 });
    } catch (error) {
      return json({ ok: false, error: error instanceof Error ? error.message : String(error) }, { status: 500 });
    }
  }
};
