# Sub2API 私有 Fork 工作流

这个仓库的长期维护模式不是“直接改线上”，而是：

1. 官方上游 `Wei-Shaw/sub2api` 持续更新
2. 我们自己的私有仓库承载部署相关定制和后续功能开发
3. 本地机器负责拉上游、解决冲突、构建镜像、生成部署包
4. 服务器只负责接收构建产物并运行 `Docker Compose`

## 远端模型

- `upstream`: 官方仓库
- `origin`: 私有 GitHub 仓库

当前本地仓库已经接入 `upstream`。创建私有仓库后，把它接成 `origin`：

```bash
git remote add origin <your-private-git-url>
git push -u origin feat/private-fork-cutover
```

如果后续要建立长期部署分支，推荐保留这类分工：

- `main`: 跟随上游、保持可同步
- `deploy/private`: 私有部署与功能定制的长期分支
- 临时功能分支：每次新功能或迁移任务单独开

## 私有改动放哪里

为了减少以后跟上游合并时的冲突，私有差异优先放在这些位置：

- `deploy/docker-compose.private.yml`
- `deploy/private/`
- `deploy/scripts/`
- `deploy/migration/`
- `deploy/runbooks/`
- `docs/operations/`

尽量不要把“仅私有环境需要的运维差异”直接散落到上游已有的核心部署文件里。

## 日常节奏

每次准备更新生产前，按固定顺序走：

1. 拉取上游最新提交
2. 合并到私有长期分支
3. 解决代码和部署材料冲突
4. 本地重新构建镜像
5. 导出发布包
6. 服务器做预检、备份、切换、烟测

当前生产机 `154.26.179.199` 是 `linux/amd64`，本地发版时要显式构建：

```bash
./deploy/build_image.sh --platform linux/amd64 ...
```

## 服务器职责边界

服务器只做这些事：

- 保存 `.env` 和迁移后的运行时数据
- `docker load` 导入本地构建的镜像
- `docker compose -f deploy/docker-compose.local.yml -f deploy/docker-compose.private.yml up -d`
- 执行预检、烟测、回滚

服务器不做这些事：

- 不在生产机上跑前端/后端源码编译
- 不在生产机上直接做日常开发
- 不把“先试试再说”当成迁移策略
