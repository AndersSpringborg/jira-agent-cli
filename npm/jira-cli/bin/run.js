#!/usr/bin/env node

const { execFileSync } = require("child_process");
const path = require("path");
const os = require("os");

// Map Node.js platform/arch to our npm package names
const PLATFORMS = {
  "darwin-x64": "@888aaen/jira-cli-darwin-x64",
  "darwin-arm64": "@888aaen/jira-cli-darwin-arm64",
  "linux-x64": "@888aaen/jira-cli-linux-x64",
  "linux-arm64": "@888aaen/jira-cli-linux-arm64",
  "win32-x64": "@888aaen/jira-cli-win32-x64",
};

const platformKey = `${os.platform()}-${os.arch()}`;
const pkg = PLATFORMS[platformKey];

if (!pkg) {
  console.error(
    `Unsupported platform: ${platformKey}\n` +
      `jira-cli supports: ${Object.keys(PLATFORMS).join(", ")}`
  );
  process.exit(1);
}

// Resolve the binary from the platform-specific package
let binPath;
try {
  const binName = os.platform() === "win32" ? "jira.exe" : "jira";
  binPath = path.join(
    path.dirname(require.resolve(`${pkg}/package.json`)),
    "bin",
    binName
  );
} catch {
  console.error(
    `Could not find the jira binary for your platform (${platformKey}).\n` +
      `The optional dependency ${pkg} may not have been installed.\n` +
      `Try reinstalling: npm install @888aaen/jira-cli`
  );
  process.exit(1);
}

// Forward all arguments to the Go binary
try {
  const result = execFileSync(binPath, process.argv.slice(2), {
    stdio: "inherit",
  });
} catch (err) {
  if (err.status !== undefined) {
    process.exit(err.status);
  }
  throw err;
}
