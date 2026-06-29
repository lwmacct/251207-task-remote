# 发布维护说明

## 本地调试 npx CLI

当 CLI 尚未发布到 npm，或者需要在发布前验证当前工作区的打包结果时，先打 tarball，再通过 `npx --package` 执行。不要直接调用本地目录，因为它不能完整模拟发布包的 `bin` 行为。

```bash
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

npm pack --silent --pack-destination "$tmp_dir" >/dev/null
package_file=$(find "$tmp_dir" -name '*.tgz' -print -quit)

npx --yes --package "$package_file" 251207-task-remote version set "0.6.260629"
```

这等价于模拟发布后的调用方式：

```bash
npx --yes --prefix "$(mktemp -d)" @lwmacct/251207-task-remote version set "0.6.260629" --cwd "$(pwd)"
```

这个方式适合验证 `package.json#bin`、可执行权限、以及最终进入 npm 包的文件是否符合预期。

如果 `.task` 需要依赖一个新的 CLI 子命令，必须先发布包含该子命令的 npm 版本。等新版本在 npm 上可用后，再更新 `.task` 使用正式的 `npx --yes --prefix <tmp-dir> @lwmacct/251207-task-remote ... --cwd <project>` 调用。不要在同一个版本里让 `.task` 依赖尚未发布的 CLI 功能。
