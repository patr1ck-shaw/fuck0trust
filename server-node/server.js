import express from 'express';
import cors from 'cors';
import Database from 'better-sqlite3';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import fs from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const app = express();
const PORT = process.env.PORT || 3000;
const ADMIN_TOKEN = process.env.ADMIN_TOKEN || 'change-me-in-production';

// 数据库初始化
const dbPath = join(__dirname, 'data', 'approval.db');
fs.mkdirSync(join(__dirname, 'data'), { recursive: true });

const db = new Database(dbPath);
db.pragma('journal_mode = WAL');

// 创建表
db.exec(`
  CREATE TABLE IF NOT EXISTS devices (
    deviceId TEXT PRIMARY KEY,
    hostname TEXT,
    username TEXT,
    note TEXT,
    approved INTEGER DEFAULT 0,
    blacklisted INTEGER DEFAULT 0,
    requestedAt INTEGER,
    approvedAt INTEGER,
    lastCheckAt INTEGER
  )
`);

// 中间件
app.use(cors());
app.use(express.json());

// 日志中间件
app.use((req, res, next) => {
  const timestamp = new Date().toISOString();
  console.log(`[${timestamp}] ${req.method} ${req.path}`);
  next();
});

// ============ 客户端接口 ============

// 健康检查
app.get('/health', (req, res) => {
  res.json({ ok: true, timestamp: Date.now() });
});

// 提交审批申请
app.post('/api/request', (req, res) => {
  try {
    const { deviceId, hostname, username, note } = req.body;

    if (!deviceId) {
      return res.status(400).json({ error: '缺少 deviceId' });
    }

    // 检查是否在 24 小时内已提交
    const existing = db.prepare('SELECT requestedAt FROM devices WHERE deviceId = ?').get(deviceId);
    if (existing) {
      const elapsed = Date.now() - existing.requestedAt;
      const hours = elapsed / (1000 * 60 * 60);
      if (hours < 24) {
        const remaining = Math.ceil(24 - hours);
        return res.status(429).json({
          error: `同一设备 24 小时内只允许提交一次，请 ${remaining} 小时后再试`
        });
      }
    }

    // 插入或更新
    const stmt = db.prepare(`
      INSERT INTO devices (deviceId, hostname, username, note, requestedAt, approved, blacklisted)
      VALUES (?, ?, ?, ?, ?, 0, 0)
      ON CONFLICT(deviceId) DO UPDATE SET
        hostname = excluded.hostname,
        username = excluded.username,
        note = excluded.note,
        requestedAt = excluded.requestedAt,
        approved = 0
    `);

    stmt.run(deviceId, hostname, username, note, Date.now());

    res.json({
      success: true,
      message: '审批请求已提交，请等待管理员审批'
    });
  } catch (error) {
    console.error('提交审批失败:', error);
    res.status(500).json({ error: '服务器错误' });
  }
});

// 查询审批状态
app.get('/api/status', (req, res) => {
  try {
    const { deviceId } = req.query;

    if (!deviceId) {
      return res.status(400).json({ error: '缺少 deviceId' });
    }

    const device = db.prepare('SELECT * FROM devices WHERE deviceId = ?').get(deviceId);

    if (!device) {
      return res.json({
        ok: true,
        approved: false,
        blacklisted: false,
        record: null
      });
    }

    // 更新最后检查时间
    db.prepare('UPDATE devices SET lastCheckAt = ? WHERE deviceId = ?').run(Date.now(), deviceId);

    res.json({
      ok: true,
      approved: device.approved === 1,
      blacklisted: device.blacklisted === 1,
      record: {
        deviceId: device.deviceId,
        hostname: device.hostname,
        username: device.username,
        note: device.note,
        requestedAt: device.requestedAt,
        approvedAt: device.approvedAt
      }
    });
  } catch (error) {
    console.error('查询状态失败:', error);
    res.status(500).json({ error: '服务器错误' });
  }
});

// ============ 管理员接口 ============

// Token 验证中间件
const requireAuth = (req, res, next) => {
  const authHeader = req.headers.authorization;
  const token = authHeader && authHeader.split(' ')[1];

  if (!token || token !== ADMIN_TOKEN) {
    return res.status(401).json({ error: '未授权' });
  }

  next();
};

// 获取所有设备列表
app.get('/api/admin/devices', requireAuth, (req, res) => {
  try {
    const devices = db.prepare('SELECT * FROM devices ORDER BY requestedAt DESC').all();
    res.json({ devices });
  } catch (error) {
    console.error('获取设备列表失败:', error);
    res.status(500).json({ error: '服务器错误' });
  }
});

// 批准设备
app.post('/api/admin/approve', requireAuth, (req, res) => {
  try {
    const { deviceId } = req.body;

    if (!deviceId) {
      return res.status(400).json({ error: '缺少 deviceId' });
    }

    const stmt = db.prepare(`
      UPDATE devices
      SET approved = 1, blacklisted = 0, approvedAt = ?
      WHERE deviceId = ?
    `);

    const result = stmt.run(Date.now(), deviceId);

    if (result.changes === 0) {
      return res.status(404).json({ error: '设备不存在' });
    }

    res.json({ success: true, message: '设备已批准' });
  } catch (error) {
    console.error('批准设备失败:', error);
    res.status(500).json({ error: '服务器错误' });
  }
});

// 拒绝设备
app.post('/api/admin/deny', requireAuth, (req, res) => {
  try {
    const { deviceId } = req.body;

    if (!deviceId) {
      return res.status(400).json({ error: '缺少 deviceId' });
    }

    const stmt = db.prepare(`
      UPDATE devices
      SET approved = 0, blacklisted = 0
      WHERE deviceId = ?
    `);

    const result = stmt.run(deviceId);

    if (result.changes === 0) {
      return res.status(404).json({ error: '设备不存在' });
    }

    res.json({ success: true, message: '设备已拒绝' });
  } catch (error) {
    console.error('拒绝设备失败:', error);
    res.status(500).json({ error: '服务器错误' });
  }
});

// 拉黑设备
app.post('/api/admin/blacklist', requireAuth, (req, res) => {
  try {
    const { deviceId } = req.body;

    if (!deviceId) {
      return res.status(400).json({ error: '缺少 deviceId' });
    }

    const stmt = db.prepare(`
      UPDATE devices
      SET approved = 0, blacklisted = 1
      WHERE deviceId = ?
    `);

    const result = stmt.run(deviceId);

    if (result.changes === 0) {
      return res.status(404).json({ error: '设备不存在' });
    }

    res.json({ success: true, message: '设备已拉黑' });
  } catch (error) {
    console.error('拉黑设备失败:', error);
    res.status(500).json({ error: '服务器错误' });
  }
});

// 删除设备
app.delete('/api/admin/device/:deviceId', requireAuth, (req, res) => {
  try {
    const { deviceId } = req.params;

    const stmt = db.prepare('DELETE FROM devices WHERE deviceId = ?');
    const result = stmt.run(deviceId);

    if (result.changes === 0) {
      return res.status(404).json({ error: '设备不存在' });
    }

    res.json({ success: true, message: '设备已删除' });
  } catch (error) {
    console.error('删除设备失败:', error);
    res.status(500).json({ error: '服务器错误' });
  }
});

// 启动服务器
app.listen(PORT, '0.0.0.0', () => {
  console.log('===========================================');
  console.log(`  Fuck0Trust 审批服务器 v1.0`);
  console.log('===========================================');
  console.log(`  监听地址: http://0.0.0.0:${PORT}`);
  console.log(`  数据库路径: ${dbPath}`);
  console.log(`  管理员 Token: ${ADMIN_TOKEN}`);
  console.log('===========================================');
  console.log('  提示: 请在生产环境中修改 ADMIN_TOKEN');
  console.log('===========================================');
});

// 优雅关闭
process.on('SIGINT', () => {
  console.log('\n正在关闭服务器...');
  db.close();
  process.exit(0);
});
