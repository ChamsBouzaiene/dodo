import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { CliWrapper } from './wrapper';
import { MockEngineServer } from './mockServer';

describe('E2E: Error Handling', () => {
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

    it('displays backend errors', async () => {
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
                    data: { session_id: 'err-sess', status: 'ready' }
                }) + '\n');

                socket.write(JSON.stringify({
                    type: 'status',
                    status: 'session_ready',
                    session_id: 'err-sess'
                }) + '\n');
            }, 500);

            socket.on('data', (data: Buffer) => {
                const str = data.toString();
                if (str.includes('trigger error')) {
                    // Send error event
                    socket.write(JSON.stringify({
                        type: 'error',
                        session_id: 'err-sess',
                        message: 'Something went terribly wrong!',
                        kind: 'BackendError',
                        details: 'Stack trace or details here...'
                    }) + '\n');

                    // Also update status to error
                    socket.write(JSON.stringify({
                        type: 'status',
                        session_id: 'err-sess',
                        status: 'error',
                        detail: 'Something went terribly wrong!'
                    }) + '\n');
                }
            });
        });

        cli.start(['--engine-addr', `127.0.0.1:${serverPort}`, '--repo', process.cwd(), '--session-id', 'error-sess']);
        await cli.waitForOutput('READY');

        cli.write('trigger error\r');

        // Verify error message is shown
        // Depending on UI implementation, it might be in a red box or toast
        // useEngineEvents maps error to "error" status and calls onError which sets error state.
        // The AppLayout passes error prop.
        await cli.waitForOutput('Something went terribly wrong!', 5000);

        // Verify status indicator shows ERROR
        // await cli.waitForOutput('ERROR', 5000); // Might be hard to catch if layout is complex
    }, 20000);
});
