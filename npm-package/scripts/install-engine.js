#!/usr/bin/env node

import https from 'https';
import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import os from 'os';
import { createRequire } from 'module';

const require = createRequire(import.meta.url);

// Dependencies that might not be installed yet during CI bootstrap
let tar, AdmZip;
try {
    tar = await import('tar');
    AdmZip = (await import('adm-zip')).default;
} catch (e) {
    // Graceful fallback or ignore if just bootstrapping
}

const REPO_OWNER = 'ChamsBouzaiene';
const REPO_NAME = 'dodo';
const BINARY_NAME = 'dodo-engine';

function getPlatform() {
    const platform = os.platform();
    switch (platform) {
        case 'darwin': return 'darwin';
        case 'linux': return 'linux';
        case 'win32': return 'windows';
        default: throw new Error(`Unsupported platform: ${platform}`);
    }
}

function getArch() {
    const arch = os.arch();
    switch (arch) {
        case 'x64': return 'amd64';
        case 'arm64': return 'arm64';
        default: throw new Error(`Unsupported architecture: ${arch}`);
    }
}

function getBinaryDir() {
    const homeDir = os.homedir();
    return path.join(homeDir, '.dodo', 'bin');
}

function getBinaryPath() {
    const dir = getBinaryDir();
    const ext = os.platform() === 'win32' ? '.exe' : '';
    return path.join(dir, BINARY_NAME + ext);
}

async function getLatestRelease() {
    const pkg = require('../package.json');
    return `v${pkg.version}`;
}

function downloadFile(url) {
    return new Promise((resolve, reject) => {
        const makeRequest = (url) => {
            https.get(url, { headers: { 'User-Agent': 'dodo-ai-installer' } }, (response) => {
                if (response.statusCode === 302 || response.statusCode === 301) {
                    makeRequest(response.headers.location);
                    return;
                }
                if (response.statusCode !== 200) {
                    reject(new Error(`Failed to download: ${response.statusCode}`));
                    return;
                }
                const chunks = [];
                response.on('data', chunk => chunks.push(chunk));
                response.on('end', () => resolve(Buffer.concat(chunks)));
                response.on('error', reject);
            }).on('error', reject);
        };
        makeRequest(url);
    });
}

async function install() {
    const platform = getPlatform();
    const arch = getArch();
    const version = await getLatestRelease();

    console.log(`Installing dodo-engine ${version} for ${platform}-${arch}...`);

    const ext = platform === 'windows' ? 'zip' : 'tar.gz';
    const assetName = `dodo-engine_${platform}_${arch}.${ext}`;
    const downloadUrl = `https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${assetName}`;

    const binaryDir = getBinaryDir();
    const binaryPath = getBinaryPath();

    // Create directory
    fs.mkdirSync(binaryDir, { recursive: true });

    console.log(`Downloading from ${downloadUrl}...`);

    try {
        const data = await downloadFile(downloadUrl);

        if (platform === 'windows') {
            // Handle zip for Windows
            const AdmZip = require('adm-zip');
            const zip = new AdmZip(data);
            zip.extractAllTo(binaryDir, true);
        } else {
            // Handle tar.gz for Unix
            const tempFile = path.join(os.tmpdir(), assetName);
            fs.writeFileSync(tempFile, data);
            execSync(`tar -xzf "${tempFile}" -C "${binaryDir}"`, { stdio: 'inherit' });
            fs.unlinkSync(tempFile);
        }

        // Make executable on Unix
        if (platform !== 'windows') {
            fs.chmodSync(binaryPath, 0o755);
        }

        console.log(`Successfully installed dodo-engine to ${binaryPath}`);
    } catch (error) {
        console.error(`Failed to install dodo-engine: ${error.message}`);
        console.error('You may need to install it manually from:');
        console.error(`https://github.com/${REPO_OWNER}/${REPO_NAME}/releases`);
        process.exit(1);
    }
}

// Check if already installed with correct version
const binaryPath = getBinaryPath();
if (fs.existsSync(binaryPath)) {
    console.log('dodo-engine already installed.');
    process.exit(0);
}

install().catch(err => {
    console.error(err);
    process.exit(1);
});
