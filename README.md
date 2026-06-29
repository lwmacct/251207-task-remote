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
