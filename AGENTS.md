# AGENTS.md

## 发布兼容规则

- 本仓库的 `.task` 通过 `.task/env.yml` 使用 `node src/index.js`，可以直接调用本地已实现但尚未发布的 CLI 能力。
- 其他项目的 `.task` 通过 `npx --yes @lwmacct/251207-task-remote` 调用 CLI，只能依赖已经发布到 npm 的能力。
- 新增 CLI 能力时，先在本仓库实现并验证；其他项目必须等包含该能力的 npm 版本发布后再接入。
- 保持发布包的 bin 名为 `251207-task-remote`，与非 scope 包名一致。
- 使用 `.task/env.yml` 选择 CLI 命令：本仓库使用 `node src/index.js`，其他项目使用 `npx --yes @lwmacct/251207-task-remote`。
- `version set` 默认应更新所有支持的版本文件；只有需要限制目标时才使用 `--type npm` 或 `--type python`。
- CI 缓存不要修改真实 lockfile。需要忽略根包版本变化时，使用 `lock normalize` 或 `lock hash` 生成派生内容作为 cache key。
- Taskfile 只做编排。只有复杂、可复用的逻辑才放进 CLI，例如版本计算。
