import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import fs from 'node:fs';
import { debugLog } from '../../utils/debugLogger.js';

vi.mock('node:fs', () => ({
    default: {
        promises: {
            appendFile: vi.fn().mockResolvedValue(undefined),
        },
        writeFileSync: vi.fn(),
    },
    promises: {
        appendFile: vi.fn().mockResolvedValue(undefined),
    },
    appendFileSync: vi.fn(),
    writeFileSync: vi.fn(),
}));

describe('DebugLogger', () => {
    const originalEnv = process.env;

    beforeEach(() => {
        vi.resetAllMocks();
        process.env = { ...originalEnv };
    });

    afterEach(() => {
        process.env = originalEnv;
    });

    it('does not log when DODO_DEBUG is not set', async () => {
        process.env.DODO_DEBUG = undefined;
        await debugLog.log({ category: 'event', component: 'test', message: 'hello' });
        expect(fs.promises.appendFile).not.toHaveBeenCalled();
    });

    it('does not log when DODO_DEBUG is false', async () => {
        process.env.DODO_DEBUG = 'false';
        await debugLog.log({ category: 'event', component: 'test', message: 'hello' });
        expect(fs.promises.appendFile).not.toHaveBeenCalled();
    });

    it('logs when DODO_DEBUG is true', async () => {
        process.env.DODO_DEBUG = 'true';
        await debugLog.log({ category: 'event', component: 'test', message: 'hello' });
        expect(fs.promises.appendFile).toHaveBeenCalled();

        const callArgs = vi.mocked(fs.promises.appendFile).mock.calls[0];
        const logContent = callArgs[1] as string;
        const entry = JSON.parse(logContent);

        expect(entry).toMatchObject({
            category: 'event',
            component: 'test',
            message: 'hello',
        });
        expect(entry.timestamp).toBeDefined();
    });

    it('clears log file', () => {
        debugLog.clear();
        expect(fs.writeFileSync).toHaveBeenCalledWith('/tmp/dodo_debug.log', '');
    });

    it('logs events correctly', async () => {
        process.env.DODO_DEBUG = 'true';
        await debugLog.event('MyComponent', 'customEvent', { foo: 'bar' });

        const entry = JSON.parse(vi.mocked(fs.promises.appendFile).mock.calls[0][1] as string);
        expect(entry).toMatchObject({
            category: 'event',
            component: 'MyComponent',
            message: 'Received event: customEvent',
            data: { foo: 'bar' }
        });
    });

    it('logs safe command data', async () => {
        process.env.DODO_DEBUG = 'true';
        await debugLog.command('MyComponent', 'save', { id: 1 });

        const entry = JSON.parse(vi.mocked(fs.promises.appendFile).mock.calls[0][1] as string);
        expect(entry).toMatchObject({
            category: 'command',
            component: 'MyComponent',
            message: 'Sent command: save',
            data: { id: 1 }
        });
    });
});
