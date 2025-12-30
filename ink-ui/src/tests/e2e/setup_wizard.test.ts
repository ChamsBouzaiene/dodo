import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { CliWrapper } from './wrapper';
import { MockEngineServer } from './mockServer';

describe('E2E: Setup Wizard', () => {
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

    it('completes setup flow and saves config', async () => new Promise<void>((resolve, reject) => {
        // Use a Promise to handle the async nature of the socket data assertion
        // OR simpler: use a variable captured in test scope
        // valid assertion logic inside the socket listener.

        let configSaved = false;

        cli = new CliWrapper({
            rows: 60,
            env: {
                DODO_ENGINE_PORT: serverPort.toString(),
            }
        });

        mockServer.on('connection', (socket) => {
            // Send setup_required immediately or after delay
            setTimeout(() => {
                socket.write(JSON.stringify({
                    type: 'setup_required',
                    session_id: 'setup-sess'
                }) + '\n');
            }, 500);

            socket.on('data', (data: Buffer) => {
                const str = data.toString();
                // Check if we received save_config
                try {
                    // Start looking for the JSON payload. It might be partial or mixed with prompts?
                    // The client sends JSON followed by newline
                    const lines = str.split('\n').filter((l: string) => l.trim().length > 0);
                    for (const line of lines) {
                        try {
                            const msg = JSON.parse(line);
                            if (msg.type === 'save_config') {
                                expect(msg.config).toBeDefined();
                                expect(msg.config.llm_provider).toBe('openai');
                                expect(msg.config.api_key).toBe('sk-test-key');
                                configSaved = true;
                            }
                        } catch (e) {
                            // ignore non-json
                        }
                    }
                } catch (e) {
                    // parsing error
                }
            });
        });

        (async () => {
            try {
                cli.start(['--engine-addr', `127.0.0.1:${serverPort}`, '--repo', process.cwd(), '--session-id', 'setup-sess']);

                // Wait for Intro
                await cli.waitForOutput('Welcome to Dodo', 5000);

                // Wait for cooldown (SetupWizard has 500ms cooldown on mount)
                await new Promise(r => setTimeout(r, 1000));

                // Press Enter to start
                cli.write('\r');

                // Wait for Provider selection
                await cli.waitForOutput('Select your LLM Provider', 10000);
                await new Promise(r => setTimeout(r, 600)); // Cooldown
                // Default is openai (index 0). Press Enter.
                cli.write('\r');

                // Wait for Model selection
                await cli.waitForOutput('Select Default Model', 10000);
                await new Promise(r => setTimeout(r, 600)); // Cooldown
                // Press Enter (gpt-4o)
                cli.write('\r');

                // Wait for API Key
                await cli.waitForOutput('Enter your API Key', 10000);
                await new Promise(r => setTimeout(r, 600)); // Cooldown
                // Type key
                cli.write('sk-test-key\r');

                // Wait for Auto Index
                await cli.waitForOutput('Enable Auto-Indexing', 10000);
                await new Promise(r => setTimeout(r, 2000)); // Increased Cooldown for safety
                // Type 'y'
                cli.write('y');
                await new Promise(r => setTimeout(r, 500));
                // Send Enter as backup if 'y' alone didn't trigger (though it should)
                cli.write('\r');

                // Wait for Summary
                await cli.waitForOutput('Configuration Summary', 20000);
                await cli.waitForOutput('-key', 10000); // Check partial key display

                await new Promise(r => setTimeout(r, 1000)); // Increased Cooldown
                // Press Enter to Save
                cli.write('\n');

                // Wait for Saving state
                await cli.waitForOutput('Saving configuration...', 10000);

                // Wait for Completion
                await cli.waitForOutput('Setup Complete!', 10000);

                // Wait a bit for backend to receive data
                await new Promise(r => setTimeout(r, 1000));

                if (!configSaved) {
                    throw new Error('Backend did not receive save_config command');
                }

                resolve();
            } catch (err) {
                reject(err);
            }
        })();
    }), 60000);
});
