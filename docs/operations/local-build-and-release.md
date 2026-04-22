# 本地构建与发布包流程 (Local Build and Release)

本文档说明了如何本地构建 Sub2API 镜像，并将其导出为可移植的发布包（Release Bundle），以便分发到生产服务器。

## 核心流程

1. **本地构建**: 使用 `deploy/build_image.sh` 构建 Docker 镜像。
2. **导出发布包**: 使用 `deploy/scripts/export_release_bundle.sh` 生成包含镜像和元数据的 `.tar.gz` 文件。
3. **传输**: 将发布包上传到服务器。
4. **加载**: 在服务器上使用 `deploy/scripts/load_release_bundle.sh` 加载镜像。

---

## 1. 本地构建镜像

在项目根目录下运行：

```bash
./deploy/build_image.sh --image-tag 1.2.3
```

- 默认镜像名: `sub2api-private`
- 默认标签: `local`
- 常用参数:
    - `--image-tag <TAG>`: 指定版本号（建议使用语义化版本或 Git Commit Hash）。
    - `--platform <PLATFORM>`: 指定目标平台（如 `linux/amd64`）。

## 2. 导出发布包

构建完成后，运行导出脚本：

```bash
./deploy/scripts/export_release_bundle.sh --image-tag 1.2.3
```

这将生成 `./release/sub2api-release-1.2.3.tar.gz`。

**发布包包含:**
- `image.tar`: `docker save` 产生的镜像存档。
- `manifest.json`: 包含版本、构建日期、镜像摘要（SHA256）等元数据的清单。

## 3. 在服务器上加载

上传发布包到服务器后，执行：

```bash
./deploy/scripts/load_release_bundle.sh /path/to/sub2api-release-1.2.3.tar.gz
```

脚本会自动：
- 解压发布包。
- 校验镜像的 SHA256 摘要（如果支持）。
- 执行 `docker load` 导入镜像。

## 4. 冒烟测试

你可以使用以下脚本验证导出/加载流程是否正常工作：

```bash
./deploy/tests/release_bundle_smoke.sh
```

---

## 为什么使用这种方式？

1. **生产环境解耦**: 生产服务器不需要安装 Go、Node.js 或 PNPM，也不需要拉取源代码。
2. **确定性**: 发布包内含 manifest，确保传输后的镜像与本地构建的一致。
3. **离线支持**: 即使服务器无法访问外部 Docker Registry，也可以通过这种方式更新。
4. **安全性**: 敏感配置（`.env`）不包含在发布包中，保持制品库的清洁。
