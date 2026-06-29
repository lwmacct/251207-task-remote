#!/usr/bin/env node

const versionCommand = require("./version");

function usage() {
  console.error("Usage: 251207-task-remote version <next|set> [...args]");
}

function main(argv = process.argv.slice(2)) {
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
