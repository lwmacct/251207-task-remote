#!/usr/bin/env node

const versionCommand = require("./version");
const lockCommand = require("./lock");
const gitCommand = require("./git");

function usage() {
  console.error("Usage: 251207-task-remote <version|lock|git> [...args]");
  console.error("Example: 251207-task-remote version set 1.2.3");
}

function main(argv = process.argv.slice(2)) {
  if (argv[0] === "git") {
    gitCommand(argv.slice(1));
    return;
  }

  if (argv[0] === "lock") {
    lockCommand(argv.slice(1));
    return;
  }

  if (argv[0] === "version") {
    versionCommand(argv.slice(1));
    return;
  }

  usage();
  process.exitCode = 1;
}

if (require.main === module) {
  try {
    main();
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}

module.exports = {
  main,
};
