import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { CliWrapper } from './wrapper';
import { MockEngineServer } from './mockServer';

describe('E2E: Session Management', () => {
    let mockServer: MockEngineServer;
    let cli: CliWrapper;
    let serverPort: number;

    beforeAll(async () => {
        mockServer = new MockEngineServer();
        serverPort = await mockServer.start();
    });

    afterAll(async () => {
        if (cli) cli.stop();
        if (mockServer) await mockServer.stop();
    });

    it('updates session context and handles reloading', async () => {
        cli = new CliWrapper({
            rows: 60,
            env: {
                DODO_ENGINE_PORT: serverPort.toString(),
            }
        });

        // Handshake
        mockServer.on('connection', (socket) => {
            setTimeout(() => {
                socket.write(JSON.stringify({
                    type: 'session_update',
                    data: { session_id: 'sess-1', status: 'ready' }
                }) + '\n');

                socket.write(JSON.stringify({
                    type: 'status',
                    status: 'session_ready',
                    session_id: 'sess-1'
                }) + '\n');
            }, 500);

            socket.on('data', (data: Buffer) => {
                const str = data.toString();
                if (str.includes('reload session')) {
                    // Simulate reload by sending new session ready with different ID
                    setTimeout(() => {
                        socket.write(JSON.stringify({
                            type: 'status',
                            status: 'session_ready',
                            session_id: 'sess-2-reloaded'
                        }) + '\n');

                        // And maybe a context event to show it persists or is new
                        socket.write(JSON.stringify({
                            type: 'context',
                            session_id: 'sess-2-reloaded',
                            kind: 'file_read',
                            description: 'Reading src/index.ts',
                            before: 0,
                            after: 100
                        }) + '\n');
                    }, 500);
                }
            });
        });

        cli.start(['--engine-addr', `127.0.0.1:${serverPort}`, '--repo', process.cwd(), '--session-id', 'mgmt-session']);
        await cli.waitForOutput('READY');

        // Initial session ID check (footer)
        // Footer usually shows "Session: sess-1" or part of it
        await cli.waitForOutput('sess-1', 5000);

        cli.write('reload session\r');

        // Verify new session ID is shown
        await cli.waitForOutput('sess-2-reloaded', 5000);

        // Verify context event is displayed
        // "Reading src/index.ts"
        await cli.waitForOutput('Reading src/index.ts', 5000);
    }, 20000);
});
