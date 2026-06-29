# AGENTS.md

## Release Compatibility

- Do not update `.task` files to depend on a CLI subcommand or behavior that has not been published to npm yet.
- When adding a new CLI feature, first implement and release it in `@lwmacct/251207-task-remote`.
- Only after that release is available on npm should `.task` be changed to call the new CLI feature.
- In `.task`, call the CLI through a temporary `--prefix` and pass the project path with `--cwd`; this avoids npx resolving the current package when the repository has the same name as the CLI package.
- `version set` should update all supported version files by default. Use `--type npm` or `--type python` only when intentionally limiting the target.
- Taskfile entries should stay as orchestration. Move only complex, reusable logic into the CLI, such as version calculation.
