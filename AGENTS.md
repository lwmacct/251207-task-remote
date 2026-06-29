# AGENTS.md

## Release Compatibility

- Do not update `.task` files to depend on a CLI subcommand or behavior that has not been published to npm yet.
- When adding a new CLI feature, first implement and release it in `@lwmacct/251207-task-remote`.
- Only after that release is available on npm should `.task` be changed to call the new CLI feature.
- Taskfile entries should stay as orchestration. Move only complex, reusable logic into the CLI, such as version calculation.
