# 251207 Task Remote

常用 Taskfile 逻辑的 Go CLI。

## 使用

```bash
go install github.com/lwmacct/251207-task-remote@latest
```

确保 Go bin 目录在 `PATH` 中：

```bash
export PATH="$HOME/go/bin:$PATH"
```

本仓库开发时可直接运行：

```bash
go run . --help
```

命令用法以 CLI help 为准：

```bash
251207-task-remote --help
```

## 能力

- 版本递增与版本文件更新
- lockfile 派生内容生成，用于 CI cache key，不修改真实 lockfile
- Git 默认分支、临时分支名、备份分支名等辅助输出

当前支持的版本文件：

- npm: `package.json`、`package-lock.json`
- Python: `pyproject.toml`

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
