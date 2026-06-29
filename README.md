# 251207 Task Remote

常用 Taskfile 逻辑的 npm CLI。

## 使用

```bash
npx --yes @lwmacct/251207-task-remote version set v1.2.3
```

## 版本

更新当前项目中所有支持的版本文件：

```bash
npx --yes @lwmacct/251207-task-remote version set v1.2.3
```

当前支持：

- npm: `package.json`、`package-lock.json`
- Python: `pyproject.toml`

只更新指定类型：

```bash
npx --yes @lwmacct/251207-task-remote version set v1.2.3 --type npm
npx --yes @lwmacct/251207-task-remote version set v1.2.3 --type python
```

计算下一个版本：

```bash
npx --yes @lwmacct/251207-task-remote version next 3
```

## Lockfile 缓存 Key

生成忽略根包版本信息的 npm lockfile 派生文件：

```bash
npx --yes @lwmacct/251207-task-remote lock normalize --type npm --output tmp/cache/package-lock.deps.json
```

直接输出派生内容的 SHA256：

```bash
npx --yes @lwmacct/251207-task-remote lock hash --type npm
```

这个能力用于 CI cache key，不会修改真实的 `package-lock.json`。

手动触发的示例 workflow 见 `.github/workflows/cache-example.yml`。

## Git 辅助输出

这些命令只输出值，不修改 git 状态。Taskfile 可以用它们减少 shell 字符串解析。

输出默认分支名：

```bash
npx --yes @lwmacct/251207-task-remote git default-branch
```

生成临时开发分支名：

```bash
npx --yes @lwmacct/251207-task-remote git dev-branch-name
npx --yes @lwmacct/251207-task-remote git dev-branch-name feature-a
```

生成备份分支名：

```bash
npx --yes @lwmacct/251207-task-remote git backup-branch-name main
```

## Taskfile 远程更新

Task 的 remote taskfiles 仍是实验特性，使用前需要启用：

```bash
export TASK_X_REMOTE_TASKFILES=1
```

远程 Taskfile 默认会缓存到本地 `.task` 目录，但默认缓存有效期是 `0s`，也就是每次执行都会尝试下载最新内容。

常用更新方式：

```bash
# 正常执行，默认会尝试获取最新远程 Taskfile
task <task-name>

# 强制忽略缓存并重新下载远程 Taskfile
task --download <task-name>

# 清空所有 remote taskfiles 缓存
task --clear-cache
```

如果配置了缓存有效期，例如：

```bash
task --expiry 24h <task-name>
```

在缓存过期前，Task 会继续使用本地缓存。需要立即更新时，使用 `--download`；需要彻底清理缓存时，使用 `--clear-cache`。

离线场景下可以使用：

```bash
task --offline <task-name>
```

此时 Task 会使用已有缓存，即使缓存已经过期。

不要运行不可信来源的远程 Taskfile。首次运行或远程内容校验和变化时，Task 会提示确认；非交互环境可使用 `--yes` 或配置 trusted hosts，但更推荐固定远程引用的 `ref`。
