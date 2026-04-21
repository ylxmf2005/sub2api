# 本地构建与发布包流程

## 目标

本地完成镜像构建，生成一个可复制到服务器的 release bundle。服务器只负责：

- `docker load`
- 放置 compose / env 文件
- `docker compose up -d`

## 基本命令

构建私有镜像：

```bash
./deploy/build_image.sh \
  --image-name local/sub2api-private \
  --image-tag 2026-04-17 \
  --platform linux/amd64
```

导出发布包：

```bash
./deploy/scripts/export_release_bundle.sh \
  --image local/sub2api-private:2026-04-17 \
  --release sub2api-2026-04-17
```

生成结果默认放在 `release/` 下，包含：

- `image.tar`
- `image.tar.sha256`
- `manifest.env`
- compose 文件副本
- `.env` 模板副本

## 约束

- release bundle 里不放真实生产密钥
- 真实 `.env` 只在服务器上维护
- 每个 release bundle 都要带清晰版本号，便于回滚
- 给生产包显式指定平台；当前 `154.26.179.199` 这台服务器是 `linux/amd64`

## 推荐命名

- 镜像：`local/sub2api-private:<date-or-version>`
- 发布包：`sub2api-<date-or-version>`

## 上线前最少确认

- 本地镜像标签正确
- 发布包 checksum 已生成
- `deploy/docker-compose.private.yml` 已包含正确的私有镜像引用方式
- 服务器上的 `.env` 与当前 release 兼容
