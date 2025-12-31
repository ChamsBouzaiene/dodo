import { defineConfig } from 'tsup';
import { readFileSync } from 'fs';

const pkg = JSON.parse(readFileSync('./package.json', 'utf-8'));

export default defineConfig({
    entry: ['src/index.tsx'],
    format: ['esm'],
    outDir: 'dist',
    clean: true,
    define: {
        'process.env.CLI_VERSION': JSON.stringify(pkg.version),
    },
});
