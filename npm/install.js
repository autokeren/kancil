// kancil npm installer — downloads the prebuilt binary from GitHub Releases.
const { execSync } = require("child_process");
const https = require("https");
const fs = require("fs");
const path = require("path");
const os = require("os");
const { createGzip } = require("zlib");

const VERSION = "0.1.0";
const REPO = "autokeren/kancil";

function platformZip() {
  const plat = os.platform();
  const arch = os.arch();
  const map = {
    darwin: { x64: "darwin-amd64", arm64: "darwin-arm64" },
    linux: { x64: "linux-amd64", arm64: "linux-amd64" },
    win32: { x64: "windows-amd64" },
  };
  const p = map[plat];
  if (!p || !p[arch]) {
    throw new Error(`kancil: unsupported platform ${plat}-${arch}`);
  }
  return `kancil-${p[arch]}.zip`;
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    https.get(url, (res) => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        return download(res.headers.location, dest).then(resolve).catch(reject);
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`kancil: download failed ${res.statusCode}`));
      }
      res.pipe(file);
      file.on("finish", () => file.close(resolve));
    }).on("error", reject);
  });
}

async function main() {
  const zip = platformZip();
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${zip}`;
  const dest = path.join(__dirname, zip);
  process.stdout.write(`  🦌 downloading kancil binary (${zip})…\n`);
  await download(url, dest);
  process.stdout.write(`  🦌 extracted: kancil\n`);

  // unzip via tar (cross-platform safe on most systems) or powershell on win
  const binDir = path.join(__dirname, "bin");
  fs.mkdirSync(binDir, { recursive: true });
  const binPath = path.join(binDir, "kancil" + (os.platform() === "win32" ? ".exe" : ""));
  if (os.platform() === "win32") {
    execSync(`powershell -c "Expand-Archive -Force '${dest}' -DestinationPath '${binDir}'"`);
  } else {
    execSync(`cd '${binDir}' && unzip -o '${dest}'`);
  }
  fs.chmodSync(path.join(binDir, "kancil" + (os.platform() === "win32" ? ".exe" : "")), 0o755);
  process.stdout.write(`  🦌 kancil installed! Run: kancil\n`);
}

main().catch((e) => { console.error(e.message); process.exit(1); });
