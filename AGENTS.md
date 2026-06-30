# AGENTS.md

## 发布兼容规则

- 本仓库的 `.task` 通过 `.task/env.yml` 使用 `go run .`，可以直接调用本地已实现但尚未发布的 CLI 能力。
- 其他项目的 `.task` 通过 `251207-task-remote` 调用 CLI；如果命令不存在，则先执行 `go install github.com/lwmacct/251207-task-remote@latest`。
- 新增 CLI 能力时，先在本仓库实现并验证；其他项目必须等包含该能力的 Git tag 发布后再接入。
- 保持 Go 安装路径为 `go install github.com/lwmacct/251207-task-remote@latest`，二进制名为 `251207-task-remote`。
- `bump set` 默认应更新所有支持的版本文件；只有需要限制目标时才使用 `--type npm` 或 `--type python`。
- CI 缓存不要修改真实 lockfile。需要忽略根包版本变化时，使用 `lock normalize` 或 `lock hash` 生成派生内容作为 cache key。
- Taskfile 只做编排。只有复杂、可复用的逻辑才放进 CLI，例如版本计算、默认分支解析、分支名生成。
