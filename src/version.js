const { spawnSync } = require("node:child_process");
const { existsSync, readFileSync, writeFileSync } = require("node:fs");
const path = require("node:path");

function parseGlobalArgs(argv) {
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

  return { args, cwd };
}

function parseSetArgs(argv) {
  const { args, cwd } = parseGlobalArgs(argv);
  const options = {
    version: null,
    cwd,
    type: "all",
  };
  const positionals = [];

  for (let i = 0; i < args.length; i += 1) {
    if (args[i] === "--type") {
      const value = args[i + 1];
      if (!value) {
        throw new Error("--type requires a value");
      }
      options.type = value;
      args.splice(i, 2);
      i -= 1;
      continue;
    }

    positionals.push(args[i]);
  }

  options.version = positionals[0];
  return options;
}

function parseNextArgs(argv) {
  const { args, cwd } = parseGlobalArgs(argv);
  const options = {
    level: "3",
    cwd,
    tag: null,
    branch: null,
    date: null,
  };
  const positionals = [];

  for (let i = 0; i < args.length; i += 1) {
    if (args[i] === "--tag") {
      const value = args[i + 1];
      if (!value) {
        throw new Error("--tag requires a value");
      }
      options.tag = value;
      args.splice(i, 2);
      i -= 1;
      continue;
    }

    if (args[i] === "--branch") {
      const value = args[i + 1];
      if (!value) {
        throw new Error("--branch requires a value");
      }
      options.branch = value;
      args.splice(i, 2);
      i -= 1;
      continue;
    }

    if (args[i] === "--date") {
      const value = args[i + 1];
      if (!value) {
        throw new Error("--date requires a YYMMDD value");
      }
      options.date = value;
      args.splice(i, 2);
      i -= 1;
      continue;
    }

    positionals.push(args[i]);
  }

  options.level = positionals[0] || options.level;
  return options;
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

function read(command, args, cwd) {
  const result = spawnSync(command, args, {
    cwd,
    encoding: "utf8",
    shell: process.platform === "win32",
  });

  if (result.error) {
    throw result.error;
  }

  if (typeof result.status === "number" && result.status !== 0) {
    return "";
  }

  return result.stdout.trim();
}

function shanghaiDateYYMMDD() {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone: "Asia/Shanghai",
    year: "2-digit",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(new Date());
  const values = Object.fromEntries(parts.map((part) => [part.type, part.value]));
  return `${values.year}${values.month}${values.day}`;
}

function latestTag(cwd) {
  return read("git", ["tag", "--sort=-v:refname"], cwd).split(/\r?\n/).filter(Boolean)[0] || "v0.0.0";
}

function currentBranch(cwd) {
  return read("git", ["rev-parse", "--abbrev-ref", "HEAD"], cwd) || "main";
}

function latestDevMinor(cwd) {
  const tags = read("git", ["tag", "--sort=-v:refname"], cwd).split(/\r?\n/).filter((tag) => /^v0\./.test(tag));
  if (tags.length === 0) {
    return 0;
  }

  return Number.parseInt(tags[0].replace(/^v/, "").split(".")[1], 10) || 0;
}

function parseVersion(tag) {
  const [major = "0", minor = "0", patch = "0"] = tag.replace(/^v/, "").split(".");
  return {
    major: Number.parseInt(major, 10) || 0,
    minor: Number.parseInt(minor, 10) || 0,
    patch: Number.parseInt(patch, 10) || 0,
  };
}

function nextVersion(argv) {
  const options = parseNextArgs(argv);
  const tag = options.tag || latestTag(options.cwd);
  const branch = options.branch || currentBranch(options.cwd);
  const { major, minor, patch } = parseVersion(tag);

  if (branch.startsWith("dev/") || major === 0) {
    const nextMinor = latestDevMinor(options.cwd) + 1;
    return `v0.${nextMinor}.${options.date || shanghaiDateYYMMDD()}`;
  }

  switch (options.level) {
    case "1":
    case "major":
      return `v${major + 1}.0.0`;
    case "2":
    case "minor":
      return `v${major}.${minor + 1}.0`;
    default:
      return `v${major}.${minor}.${patch + 1}`;
  }
}

function setVersion(argv) {
  const { version, cwd, type } = parseSetArgs(argv);
  if (!version) {
    throw new Error("version set requires a version");
  }

  const normalizedVersion = version.replace(/^v/, "");
  const resolvedTypes = resolveVersionTypes(type, cwd);

  for (const resolvedType of resolvedTypes) {
    if (resolvedType === "npm") {
      setNpmVersion(normalizedVersion, cwd);
      continue;
    }

    if (resolvedType === "python") {
      setPythonVersion(normalizedVersion, cwd);
      continue;
    }

    throw new Error(`Unsupported version type: ${resolvedType}`);
  }
}

function resolveVersionTypes(type, cwd) {
  if (type === "npm" || type === "python") {
    return [type];
  }

  if (type !== "all" && type !== "auto") {
    throw new Error(`Unsupported version type: ${type}`);
  }

  const types = [];
  if (existsSync(path.join(cwd, "package.json"))) {
    types.push("npm");
  }

  if (existsSync(path.join(cwd, "pyproject.toml"))) {
    types.push("python");
  }

  if (types.length === 0) {
    throw new Error("No supported version file found. Use --type npm or --type python.");
  }

  return types;
}

function setNpmVersion(version, cwd) {
  const packageJson = path.join(cwd, "package.json");
  if (!existsSync(packageJson)) {
    throw new Error(`package.json not found in ${cwd}`);
  }

  run("npm", ["version", version, "--no-git-tag-version", "--allow-same-version"], cwd);
  run("npm", ["install", "--package-lock-only", "--ignore-scripts"], cwd);
}

function setPythonVersion(version, cwd) {
  const pyprojectPath = path.join(cwd, "pyproject.toml");
  if (!existsSync(pyprojectPath)) {
    throw new Error(`pyproject.toml not found in ${cwd}`);
  }

  const content = readFileSync(pyprojectPath, "utf8");
  const nextContent = content.replace(/^version\s*=\s*["'][^"']*["']/m, `version = "${version}"`);
  if (nextContent === content) {
    throw new Error(`version field not found in ${pyprojectPath}`);
  }

  writeFileSync(pyprojectPath, nextContent);
}

function versionCommand(argv) {
  if (argv[0] === "next") {
    console.log(nextVersion(argv.slice(1)));
    return;
  }

  if (argv[0] === "set") {
    setVersion(argv.slice(1));
    return;
  }

  throw new Error("Usage: 251207-task-remote version set <version> [--type <all|npm|python>]");
}

module.exports = versionCommand;
