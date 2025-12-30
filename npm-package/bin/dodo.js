#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');

function getBinaryPath() {
    const homeDir = os.homedir();
    const ext = os.platform() === 'win32' ? '.exe' : '';
    return path.join(homeDir, '.dodo', 'bin', 'dodo-engine' + ext);
}

const binaryPath = getBinaryPath();

if (!fs.existsSync(binaryPath)) {
    console.error('dodo-engine not found. Running installer...');
    require('../scripts/install-engine.js');
    process.exit(0);
}

// Pass all arguments to the engine
const args = process.argv.slice(2);

const child = spawn(binaryPath, args, {
    stdio: 'inherit',
    env: process.env
});

child.on('error', (err) => {
    console.error(`Failed to start dodo-engine: ${err.message}`);
    process.exit(1);
});

child.on('close', (code) => {
    process.exit(code || 0);
});
