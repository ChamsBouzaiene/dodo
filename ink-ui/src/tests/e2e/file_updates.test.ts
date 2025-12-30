import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { CliWrapper } from './wrapper';
import { MockEngineServer } from './mockServer';

describe('E2E: File Updates', () => {
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

    it('displays file changes (diffs)', async () => {
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
                    data: { session_id: 'file-sess', status: 'ready' }
                }) + '\n');

                socket.write(JSON.stringify({
                    type: 'status',
                    status: 'session_ready',
                    session_id: 'file-sess'
                }) + '\n');
            }, 500);

            socket.on('data', (data: Buffer) => {
                const str = data.toString();
                if (str.includes('edit file')) {
                    // Send edit activity with code_change
                    socket.write(JSON.stringify({
                        type: 'activity',
                        session_id: 'file-sess',
                        activity_id: 'act-edit-1',
                        activity_type: 'edit',
                        tool: 'str_replace',
                        status: 'completed', // Edits are usually instant/atomic in display
                        timestamp: new Date().toISOString(),
                        code_change: {
                            file: 'src/config.ts',
                            before: 'const debug = false;',
                            after: 'const debug = true;',
                            start_line: 10,
                            end_line: 10
                        }
                    }) + '\n');

                    // Then done
                    setTimeout(() => {
                        socket.write(JSON.stringify({
                            type: 'assistant_text',
                            session_id: 'file-sess',
                            content: 'I updated the config.',
                            final: true
                        }) + '\n');

                        socket.write(JSON.stringify({
                            type: 'session_update',
                            data: { session_id: 'file-sess', status: 'ready' }
                        }) + '\n');
                    }, 500);
                }
            });
        });

        cli.start(['--engine-addr', `127.0.0.1:${serverPort}`, '--repo', process.cwd(), '--session-id', 'file-session']);
        await cli.waitForOutput('READY');

        cli.write('edit file\r');

        // Verify file path is shown
        await cli.waitForOutput('src/config.ts', 5000);

        // Verify diff content
        // Note: CodeDiff usually shows lines preceeded by + or -
        // We look for "-const debug = false;" or similar logic depending on CodeDiff implementation
        await cli.waitForOutput('const debug = false;', 5000);
        await cli.waitForOutput('const debug = true;', 5000);

        await cli.waitForOutput('I updated the config.', 5000);
    }, 20000);
});
