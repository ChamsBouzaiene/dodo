import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { CliWrapper } from './wrapper';
import { MockEngineServer } from './mockServer';
import path from 'path';

describe('E2E: Startup', () => {
    let mockServer: MockEngineServer;
    let cli: CliWrapper;
    let serverPort: number;

    beforeAll(async () => {
        // Start Mock Engine Server
        mockServer = new MockEngineServer();
        serverPort = await mockServer.start();
    });

    afterAll(async () => {
        if (cli) cli.stop();
        if (mockServer) await mockServer.stop();
    });

    it('connects to engine and shows ready status', async () => {
        cli = new CliWrapper({
            env: {
                DODO_ENGINE_PORT: serverPort.toString(),
            }
        });

        // Mock Server Handling - Register BEFORE starting CLI
        mockServer.on('connection', (socket) => {
            console.log('[TEST] Mock Server: Client connected');
            // Wait a bit for client to be ready
            setTimeout(() => {
                console.log('[TEST] Mock Server: Writing handshake');
                // Simulate handshake/initial data
                const initialData = {
                    type: 'session_update',
                    data: {
                        session_id: 'e2e-session-123',
                        status: 'ready'
                    }
                };
                // Send data as newline delimited JSON
                const payload = JSON.stringify(initialData) + '\n';
                socket.write(payload);

                // Also send status event to resolve startSession promise
                const statusEvent = {
                    type: 'status',
                    status: 'session_ready',
                    session_id: 'e2e-session-123'
                };
                socket.write(JSON.stringify(statusEvent) + '\n', (err) => {
                    if (err) console.error('[TEST] Write error:', err);
                    else console.log('[TEST] Write success');
                });
            }, 500);
        });

        // Start CLI
        // Use 127.0.0.1 to match index.tsx debug logs and prevent IPv6 issues
        cli.start(['--engine-addr', `127.0.0.1:${serverPort}`, '--repo', process.cwd(), '--session-id', 'e2e-session-123', '--log-file', 'startup_test.log']);

        // Expect "Connecting..." then "Ready"
        // Wait for "READY" status in footer (which is typically uppercased in StatusBadge)
        await cli.waitForOutput('e2e-session-123', 10000);
        await cli.waitForOutput('READY');
    }, 15000);
});
