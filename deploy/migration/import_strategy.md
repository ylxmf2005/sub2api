# 导入策略

## 先决条件

- 已通过 `audit_server.sh` 明确源部署模式
- 已完成完整备份
- 已确认关键密钥是否可以沿用

## 按源模式选择导入路径

### 1. `docker-local`

- 最终停写后冻结旧目录
- 复制旧部署目录里的：
  - `data/`
  - `postgres_data/`
  - `redis_data/`
  - `.env`
  - compose 文件
- 在新工作目录里放入私有 overlay
- 核对 `.env` 后再启动

### 2. `docker-named`

- 导出命名卷
- 在新目标目录恢复成：
  - `data/`
  - `postgres_data/`
  - `redis_data/`
- 保证目录落位完成后再第一次 `docker compose up`

### 3. `binary`

- 先备份 `/opt/sub2api` 与 `/etc/sub2api`
- 提取 `config.yaml` 与任何环境变量
- 从旧数据库/Redis 导出真实数据
- 还原到新 Compose 目录后再启动

### 4. `mixed`

- 不做“猜一个主路径”
- 先确认谁才是当前生产流量真实来源
- 未确认前不允许切服

## 关键门槛

- 如果拿不到 `JWT_SECRET` / `TOTP_ENCRYPTION_KEY`，要把这件事明确记为切服风险
- 如果旧环境存在额外宿主机组件，例如 `datamanagementd`，必须同时迁移挂载与服务托管关系
- 所有导入动作完成前，不允许第一次启动新栈
