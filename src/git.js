const { spawnSync } = require("node:child_process");

function read(command, args) {
  const result = spawnSync(command, args, {
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

function shanghaiDate(parts) {
  const values = Object.fromEntries(new Intl.DateTimeFormat("en-US", {
    timeZone: "Asia/Shanghai",
    year: "2-digit",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    hour12: false,
  }).formatToParts(new Date()).map((part) => [part.type, part.value]));

  return parts.map((part) => values[part]).join("");
}

function defaultBranch() {
  const originHead = read("git", ["symbolic-ref", "refs/remotes/origin/HEAD"]);
  if (originHead.startsWith("refs/remotes/origin/")) {
    return originHead.replace("refs/remotes/origin/", "");
  }

  const remoteBranches = read("git", ["branch", "-r"]).split(/\r?\n/).map((line) => line.trim());
  for (const branch of ["main", "master"]) {
    if (remoteBranches.includes(`origin/${branch}`)) {
      return branch;
    }
  }

  return "main";
}

function currentBranch() {
  return read("git", ["branch", "--show-current"]);
}

function devBranchName(argv) {
  const suffix = argv.join(" ") || shanghaiDate(["hour", "minute"]);
  return `dev/${shanghaiDate(["year", "month", "day"])}-${suffix}`;
}

function backupBranchName(argv) {
  const branch = argv[0] || currentBranch();
  if (!branch) {
    return "";
  }

  const date = shanghaiDate(["year", "month", "day", "hour", "minute"]);
  return `backup/${date.slice(0, 2)}/${date.slice(2, 4)}/${date.slice(4, 6)}/${date.slice(6)}/${branch}`;
}

function gitCommand(argv) {
  if (argv[0] === "default-branch") {
    console.log(defaultBranch());
    return;
  }

  if (argv[0] === "dev-branch-name") {
    console.log(devBranchName(argv.slice(1)));
    return;
  }

  if (argv[0] === "backup-branch-name") {
    console.log(backupBranchName(argv.slice(1)));
    return;
  }

  throw new Error("Usage: 251207-task-remote git <default-branch|dev-branch-name|backup-branch-name>");
}

module.exports = gitCommand;
