#!/usr/bin/env node

import { spawn } from 'child_process';
import path from 'path';
import os from 'os';
import fs from 'fs';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// 1. Ensure Engine is Installed
function getBinaryPath() {
    const homeDir = os.homedir();
    const ext = os.platform() === 'win32' ? '.exe' : '';
    return path.join(homeDir, '.dodo', 'bin', 'dodo-engine' + ext);
}

const binaryPath = getBinaryPath();

if (!fs.existsSync(binaryPath)) {
    console.error('dodo-engine not found. Running installer...');
    try {
        const installScript = path.join(__dirname, '../scripts/install-engine.js');
        const { execSync } = require('child_process');
        execSync(`node "${installScript}"`, { stdio: 'inherit' });
    } catch (e) {
        // Ignore if failing, UI might handle it or just fail later
    }
}

// 2. Start the UI (which is now compiled into ../dist/index.js)
import('../dist/index.js').catch(err => {
    console.error('Failed to start Dodo UI:', err);
    console.error('Debug Info:');
    console.error('__dirname:', __dirname);
    try {
        const rootDir = path.join(__dirname, '..');
        console.error('Package root contents:', fs.readdirSync(rootDir));
        const distDir = path.join(rootDir, 'dist');
        if (fs.existsSync(distDir)) {
            console.error('dist contents:', fs.readdirSync(distDir));
        } else {
            console.error('dist directory does not exist');
        }
    } catch (e) {
        console.error('Failed to list directories:', e);
    }
    process.exit(1);
});
