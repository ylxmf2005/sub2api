# 生产部署目录布局

推荐把服务器工作目录固定成一套稳定结构，避免每次升级都重新找文件。

## 推荐目录

```text
sub2api-deploy/
├── .env
├── docker-compose.local.yml
├── docker-compose.private.yml
├── data/
├── postgres_data/
├── redis_data/
├── releases/
└── backups/
```

## Compose 约定

基础文件始终使用上游的本地目录版：

- `deploy/docker-compose.local.yml`

私有差异放在覆盖层：

- `deploy/docker-compose.private.yml`

实际启动命令固定成双文件：

```bash
docker compose \
  -f docker-compose.local.yml \
  -f docker-compose.private.yml \
  up -d
```

## 必须长期保留的环境值

以下值如果线上已经存在，迁移时应优先沿用，而不是重生：

- `POSTGRES_PASSWORD`
- `REDIS_PASSWORD`
- `JWT_SECRET`
- `TOTP_ENCRYPTION_KEY`
- `ADMIN_EMAIL`
- OAuth 相关密钥
- URL allowlist / proxy / 其他安全配置

## 可选联动

如果当前线上已经启用了 `datamanagementd`，需要把宿主机 socket 保留下来，并通过私有 overlay 接回容器。

## 切换原则

- 先备份
- 再导入数据或复制运行时目录
- 数据和密钥到位后，才允许第一次 `up -d`
- 烟测通过后再下线旧部署
