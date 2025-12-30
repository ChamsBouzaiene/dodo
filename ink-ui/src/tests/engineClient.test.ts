import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { EngineClient } from '../engineClient.js';
import { PassThrough } from 'node:stream';
import { debugLog } from '../utils/debugLogger.js';

// Mock debugLogger to avoid actual file writes
vi.mock('../utils/debugLogger.js', () => ({
    debugLog: {
        log: vi.fn(),
        state: vi.fn(),
        command: vi.fn(),
        error: vi.fn(),
    }
}));

describe('EngineClient', () => {
    let stdin: PassThrough;
    let stdout: PassThrough;
    let client: EngineClient;

    beforeEach(() => {
        stdin = new PassThrough();
        stdout = new PassThrough();
        client = new EngineClient(stdin, stdout);
        vi.clearAllMocks();
    });

    afterEach(() => {
        client.close();
    });

    it('emits events from stdout lines', async () => {
        const received = vi.fn();
        client.on('event', received);

        const event = { type: 'status', status: 'ready', session_id: '123' };
        stdout.write(JSON.stringify(event) + '\n');

        // Wait a tick
        await new Promise(r => setImmediate(r));

        expect(received).toHaveBeenCalledWith(event);
        expect(debugLog.log).toHaveBeenCalledWith(expect.objectContaining({ message: 'IN' }));
    });

    it('sends commands via stdin', async () => {
        const writeSpy = vi.spyOn(stdin, 'write');

        // Mock session start logic indirectly by just testing sendCommand directly
        // or ensure pendingSession isn't blocking. 
        // Let's test low level sendCommand first.
        const command: any = { type: 'start_session', repo_root: '.' };
        await client.sendCommand(command);

        expect(writeSpy).toHaveBeenCalled();
        const written = writeSpy.mock.calls[0][0] as string;
        expect(JSON.parse(written)).toEqual(command);
        expect(debugLog.log).toHaveBeenCalledWith(expect.objectContaining({ message: 'OUT' }));
    });

    it('handles session start flow', async () => {
        const startPromise = client.startSession({ repoRoot: './' });

        // Mock server response
        const sessionReadyEvent = { type: 'status', status: 'session_ready', session_id: 'test-session' };
        stdout.write(JSON.stringify(sessionReadyEvent) + '\n');

        const sessionId = await startPromise;
        expect(sessionId).toBe('test-session');
        expect(client.getSessionId()).toBe('test-session');
    });

    it('handles JSON parse errors gracefully', async () => {
        const errorSpy = vi.fn();
        client.on('error', errorSpy);

        stdout.write('invalid json\n');
        await new Promise(r => setImmediate(r));

        expect(errorSpy).toHaveBeenCalled();
        expect(debugLog.error).toHaveBeenCalled();
    });
});
