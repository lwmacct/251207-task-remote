#!/usr/bin/env node

const assert = require("node:assert/strict");
const { mkdtempSync, readFileSync, writeFileSync } = require("node:fs");
const { tmpdir } = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

function writeJson(file, value) {
  writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`);
}

function readJson(file) {
  return JSON.parse(readFileSync(file, "utf8"));
}

function runCli(args, cwd) {
  return spawnSync(process.execPath, [path.join(__dirname, "..", "src", "index.js"), ...args], {
    cwd,
    encoding: "utf8",
  });
}

function testVersionSetUpdatesNpmFilesWithoutInstallOutput() {
  const cwd = mkdtempSync(path.join(tmpdir(), "task-remote-version-"));
  writeJson(path.join(cwd, "package.json"), {
    name: "sample",
    version: "0.1.0",
    dependencies: {
      leftpad: "1.0.0",
    },
  });
  writeJson(path.join(cwd, "package-lock.json"), {
    name: "sample",
    version: "0.1.0",
    lockfileVersion: 3,
    requires: true,
    packages: {
      "": {
        name: "sample",
        version: "0.1.0",
        dependencies: {
          leftpad: "1.0.0",
        },
      },
      "node_modules/leftpad": {
        version: "1.0.0",
      },
    },
  });

  const result = runCli(["version", "set", "v1.2.3"], cwd);
  assert.equal(result.status, 0, result.stderr);
  assert.equal(result.stdout, "");
  assert.equal(result.stderr, "");

  const packageJson = readJson(path.join(cwd, "package.json"));
  const packageLock = readJson(path.join(cwd, "package-lock.json"));
  assert.equal(packageJson.version, "1.2.3");
  assert.equal(packageLock.version, "1.2.3");
  assert.equal(packageLock.packages[""].version, "1.2.3");
  assert.equal(packageLock.packages["node_modules/leftpad"].version, "1.0.0");
}

function testVersionSetRejectsUnsupportedLockfileVersion() {
  const cwd = mkdtempSync(path.join(tmpdir(), "task-remote-version-"));
  writeJson(path.join(cwd, "package.json"), {
    name: "sample",
    version: "0.1.0",
  });
  writeJson(path.join(cwd, "package-lock.json"), {
    name: "sample",
    version: "0.1.0",
    lockfileVersion: 1,
  });

  const result = runCli(["version", "set", "1.2.3"], cwd);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Unsupported package-lock\.json lockfileVersion: 1/);
  assert.equal(readJson(path.join(cwd, "package.json")).version, "0.1.0");
  assert.equal(readJson(path.join(cwd, "package-lock.json")).version, "0.1.0");
}

testVersionSetUpdatesNpmFilesWithoutInstallOutput();
testVersionSetRejectsUnsupportedLockfileVersion();
