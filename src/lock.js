const crypto = require("node:crypto");
const { existsSync, mkdirSync, readFileSync, writeFileSync } = require("node:fs");
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

function parseLockArgs(argv) {
  const { args, cwd } = parseGlobalArgs(argv);
  const options = {
    cwd,
    output: null,
    type: "npm",
  };

  for (let i = 0; i < args.length; i += 1) {
    if (args[i] === "--output") {
      const value = args[i + 1];
      if (!value) {
        throw new Error("--output requires a path");
      }
      options.output = value;
      args.splice(i, 2);
      i -= 1;
      continue;
    }

    if (args[i] === "--type") {
      const value = args[i + 1];
      if (!value) {
        throw new Error("--type requires a value");
      }
      options.type = value;
      args.splice(i, 2);
      i -= 1;
    }
  }

  return options;
}

function sortValue(value) {
  if (Array.isArray(value)) {
    return value.map(sortValue);
  }

  if (value && typeof value === "object") {
    return Object.keys(value).sort().reduce((result, key) => {
      result[key] = sortValue(value[key]);
      return result;
    }, {});
  }

  return value;
}

function normalizeNpmLock(cwd) {
  const lockPath = path.join(cwd, "package-lock.json");
  if (!existsSync(lockPath)) {
    throw new Error(`package-lock.json not found in ${cwd}`);
  }

  const lock = JSON.parse(readFileSync(lockPath, "utf8"));
  delete lock.name;
  delete lock.version;

  if (lock.packages && lock.packages[""]) {
    delete lock.packages[""].name;
    delete lock.packages[""].version;
    delete lock.packages[""].license;
    delete lock.packages[""].bin;
  }

  return `${JSON.stringify(sortValue(lock), null, 2)}\n`;
}

function normalizedLock(options) {
  if (options.type !== "npm") {
    throw new Error(`Unsupported lock type: ${options.type}`);
  }

  return normalizeNpmLock(options.cwd);
}

function normalize(argv) {
  const options = parseLockArgs(argv);
  const content = normalizedLock(options);

  if (options.output) {
    const outputPath = path.resolve(options.cwd, options.output);
    mkdirSync(path.dirname(outputPath), { recursive: true });
    writeFileSync(outputPath, content);
    return;
  }

  process.stdout.write(content);
}

function hash(argv) {
  const options = parseLockArgs(argv);
  const content = normalizedLock(options);
  console.log(crypto.createHash("sha256").update(content).digest("hex"));
}

function lockCommand(argv) {
  if (argv[0] === "normalize") {
    normalize(argv.slice(1));
    return;
  }

  if (argv[0] === "hash") {
    hash(argv.slice(1));
    return;
  }

  throw new Error("Usage: 251207-task-remote lock <normalize|hash> [--type npm] [--output path]");
}

module.exports = lockCommand;
