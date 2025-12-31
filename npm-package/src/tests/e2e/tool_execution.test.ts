import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { CliWrapper } from './wrapper';
import { MockEngineServer } from './mockServer';

describe('E2E: Tool Execution', () => {
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

    it('displays tool running and completion states', async () => {
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
                    data: { session_id: 'tool-sess', status: 'ready' }
                }) + '\n');

                socket.write(JSON.stringify({
                    type: 'status',
                    status: 'session_ready',
                    session_id: 'tool-sess'
                }) + '\n');
            }, 500);

            socket.on('data', (data: Buffer) => {
                const str = data.toString();
                if (str.includes('run tool')) {
                    // 1. Start tool
                    socket.write(JSON.stringify({
                        type: 'activity',
                        session_id: 'tool-sess',
                        activity_id: 'act-tool-1',
                        activity_type: 'tool',
                        tool: 'run_cmd',
                        command: 'ls -la',
                        status: 'started',
                        timestamp: new Date().toISOString()
                    }) + '\n');

                    setTimeout(() => {
                        // 2. Complete tool
                        socket.write(JSON.stringify({
                            type: 'activity',
                            session_id: 'tool-sess',
                            activity_id: 'act-tool-1',
                            activity_type: 'tool',
                            tool: 'run_cmd',
                            command: 'ls -la',
                            status: 'completed',
                            timestamp: new Date().toISOString(),
                            duration_ms: 1200
                        }) + '\n');

                        // 3. Mark done
                        socket.write(JSON.stringify({
                            type: 'assistant_text',
                            session_id: 'tool-sess',
                            content: 'Tool finished.',
                            final: true
                        }) + '\n');

                        socket.write(JSON.stringify({
                            type: 'session_update',
                            data: { session_id: 'tool-sess', status: 'ready' }
                        }) + '\n');
                    }, 1000);
                }
            });
        });

        cli.start(['--engine-addr', `127.0.0.1:${serverPort}`, '--repo', process.cwd(), '--session-id', 'tool-sess']);
        await cli.waitForOutput('READY');

        cli.write('run tool\r');

        // Verify running state (RUN_CMD label)
        await cli.waitForOutput('RUN_CMD', 5000);
        // Verify command is shown
        await cli.waitForOutput('$ ls -la', 5000);

        // Verify done state (finished text)
        await cli.waitForOutput('Tool finished.', 5000);
    }, 20000);
});
