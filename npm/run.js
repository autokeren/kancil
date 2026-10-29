#!/usr/bin/env node
// kancil — launcher: runs the prebuilt binary
const { spawn } = require("child_process");
const path = require("path");
const os = require("os");

const bin = path.join(__dirname, "bin", "kancil" + (os.platform() === "win32" ? ".exe" : ""));
const args = process.argv.slice(2);
const proc = spawn(bin, args, { stdio: "inherit" });
proc.on("exit", (code) => process.exit(code || 0));
