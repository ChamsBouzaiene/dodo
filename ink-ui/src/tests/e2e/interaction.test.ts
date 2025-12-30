import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { CliWrapper } from './wrapper';
import { MockEngineServer } from './mockServer';

describe('E2E: Interaction', () => {
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

    it('sends input and displays response', async () => {
        cli = new CliWrapper({
            env: {
                DODO_ENGINE_PORT: serverPort.toString(),
            }
        });

        // Handshake - Register BEFORE starting CLI
        mockServer.on('connection', (socket) => {
            setTimeout(() => {
                socket.write(JSON.stringify({
                    type: 'session_update',
                    data: { session_id: 'interact-sess', status: 'ready' }
                }) + '\n');

                socket.write(JSON.stringify({
                    type: 'status',
                    status: 'session_ready',
                    session_id: 'interact-sess'
                }) + '\n');
            }, 500);

            // Listen for user input
            socket.on('data', (data: Buffer) => {
                const str = data.toString();
                // We look for 'hello world' in the stream
                if (str.includes('hello world')) {
                    // Respond with thinking (started)
                    socket.write(JSON.stringify({
                        type: 'activity',
                        session_id: 'interact-sess',
                        activity_id: 'act1',
                        activity_type: 'thinking',
                        status: 'started',
                        timestamp: new Date().toISOString()
                    }) + '\n');

                    setTimeout(() => {
                        // Thinking done
                        socket.write(JSON.stringify({
                            type: 'activity',
                            session_id: 'interact-sess',
                            activity_id: 'act1',
                            activity_type: 'thinking',
                            status: 'completed',
                            timestamp: new Date().toISOString(),
                            duration_ms: 500
                        }) + '\n');

                        // Stream response
                        socket.write(JSON.stringify({
                            type: 'assistant_text',
                            session_id: 'interact-sess',
                            content: 'Hello there!',
                            final: true
                        }) + '\n');

                        // Set status back to ready
                        socket.write(JSON.stringify({
                            type: 'session_update',
                            data: { session_id: 'interact-sess', status: 'ready' }
                        }) + '\n');
                    }, 500);
                }
            });
        });

        // Start CLI
        cli.start(['--engine-addr', `127.0.0.1:${serverPort}`, '--repo', process.cwd(), '--session-id', 'interact-sess']);

        // Wait for ready
        await cli.waitForOutput('READY');

        // Type "hello world" + Enter
        cli.write('hello world\r');

        await cli.waitForOutput('Hello there!', 10000);
    }, 20000);
});
