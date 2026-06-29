const { spawnSync } = require("node:child_process");
const { existsSync } = require("node:fs");
const path = require("node:path");

function parseSetArgs(argv) {
  const args = [...argv];
  let cwd = process.cwd();

  for (let i = 0; i < args.length; i += 1) {
    if (args[i] === "--cwd") {
      const value = args[i + 1];
      if (!value) {
        throw new Error("--cwd requires a path");
      }
      cwd = path.resolve(value);
      args.splice(i, 2);
      i -= 1;
    }
  }

  return { version: args[0], cwd };
}

function run(command, args, cwd) {
  const result = spawnSync(command, args, {
    cwd,
    stdio: "inherit",
    shell: process.platform === "win32",
  });

  if (result.error) {
    throw result.error;
  }

  if (typeof result.status === "number" && result.status !== 0) {
    process.exit(result.status);
  }
}

function setVersion(argv) {
  const { version, cwd } = parseSetArgs(argv);
  if (!version) {
    throw new Error("version set requires a version");
  }

  const packageJson = path.join(cwd, "package.json");
  if (!existsSync(packageJson)) {
    throw new Error(`package.json not found in ${cwd}`);
  }

  const normalizedVersion = version.replace(/^v/, "");
  run("npm", ["version", normalizedVersion, "--no-git-tag-version", "--allow-same-version"], cwd);
  run("npm", ["install", "--package-lock-only", "--ignore-scripts"], cwd);
}

function versionCommand(argv) {
  if (argv[0] === "set") {
    setVersion(argv.slice(1));
    return;
  }

  throw new Error("Usage: 251207-task-remote version set <version> [--cwd <path>]");
}

module.exports = versionCommand;
