import { describe, it, expect } from 'vitest';
import { serializeCommand, Command } from '../protocol.js';

describe('Protocol Utils', () => {
    it('serializes commands correctly', () => {
        const cmd: Command = {
            type: 'user_message',
            session_id: '123',
            message: 'hello'
        };
        const serialized = serializeCommand(cmd);
        expect(serialized).toBe(JSON.stringify(cmd));

        // Verify it parses back
        expect(JSON.parse(serialized)).toEqual(cmd);
    });
});
