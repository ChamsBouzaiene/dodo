import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { CliWrapper } from './wrapper';
import { MockEngineServer } from './mockServer';

describe('E2E: Project Plan', () => {
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

    it('displays project plan when received', async () => {
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
                    data: { session_id: 'plan-sess', status: 'ready' }
                }) + '\n');

                socket.write(JSON.stringify({
                    type: 'status',
                    status: 'session_ready',
                    session_id: 'plan-sess'
                }) + '\n');
            }, 500);

            socket.on('data', (data: Buffer) => {
                const str = data.toString();
                if (str.includes('show plan')) {
                    // Send plan
                    socket.write(JSON.stringify({
                        type: 'project_plan',
                        session_id: 'plan-sess',
                        content: '1. [ ] Step One\n2. [ ] Step Two',
                        source: 'agent'
                    }) + '\n');
                }
            });
        });

        cli.start(['--engine-addr', `127.0.0.1:${serverPort}`, '--repo', process.cwd(), '--session-id', 'plan-session']);
        await cli.waitForOutput('READY');

        cli.write('show plan\r');

        // Verify Plan Header
        await cli.waitForOutput('📋 Project Plan', 5000);

        // Verify Content
        await cli.waitForOutput('Step One', 5000);
        await cli.waitForOutput('Step Two', 5000);
    }, 20000);
});
