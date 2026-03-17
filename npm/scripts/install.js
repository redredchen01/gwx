#!/usr/bin/env node
"use strict";

const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");
const https = require("https");
const os = require("os");

const VERSION = "0.3.0";
const REPO = "redredchen01/gwx";
const BIN_DIR = path.join(__dirname, "..", "bin");

function getPlatform() {
  const platform = os.platform();
  const arch = os.arch();

  const platformMap = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
  };

  const archMap = {
    x64: "amd64",
    arm64: "arm64",
  };

  const p = platformMap[platform];
  const a = archMap[arch];

  if (!p || !a) {
    console.error(`Unsupported platform: ${platform}/${arch}`);
    console.error("Please install from source: go install github.com/redredchen01/gwx/cmd/gwx@latest");
    process.exit(1);
  }

  return { os: p, arch: a, ext: platform === "win32" ? ".exe" : "" };
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const follow = (url) => {
      https.get(url, { headers: { "User-Agent": "gwx-npm-installer" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          follow(res.headers.location);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`Download failed: HTTP ${res.statusCode} from ${url}`));
          return;
        }
        const file = fs.createWriteStream(dest);
        res.pipe(file);
        file.on("finish", () => {
          file.close();
          resolve();
        });
      }).on("error", reject);
    };
    follow(url);
  });
}

async function main() {
  const { os: goos, arch, ext } = getPlatform();
  const binaryName = `gwx${ext}`;
  const assetName = `gwx_${VERSION}_${goos}_${arch}${ext}`;
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${assetName}`;

  fs.mkdirSync(BIN_DIR, { recursive: true });
  const dest = path.join(BIN_DIR, binaryName);

  console.log(`Downloading gwx v${VERSION} for ${goos}/${arch}...`);

  try {
    await download(url, dest);
    fs.chmodSync(dest, 0o755);
    console.log(`✓ Installed gwx to ${dest}`);
  } catch (err) {
    console.error(`Failed to download pre-built binary: ${err.message}`);
    console.error("");
    console.error("Falling back to 'go install'...");

    try {
      execSync("go install github.com/redredchen01/gwx/cmd/gwx@latest", { stdio: "inherit" });
      // Symlink from GOPATH/bin to our bin dir
      const gopath = execSync("go env GOPATH", { encoding: "utf-8" }).trim();
      const goBin = path.join(gopath, "bin", binaryName);
      if (fs.existsSync(goBin)) {
        fs.copyFileSync(goBin, dest);
        fs.chmodSync(dest, 0o755);
        console.log(`✓ Installed gwx via 'go install'`);
      }
    } catch (goErr) {
      console.error("Failed to install via Go. Please install manually:");
      console.error("  go install github.com/redredchen01/gwx/cmd/gwx@latest");
      process.exit(1);
    }
  }
}

main();
