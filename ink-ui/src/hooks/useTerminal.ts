import { useState, useEffect } from "react";
import { ANSI } from "../utils/ansi.js";

export type UseTerminalOptions = {
    /**
     * Whether to enter alternate screen buffer on mount.
     * @default true
     */
    enableAlternateBuffer?: boolean;

    /**
     * Initial fallback rows if stdout is not defined (e.g. testing)
     * @default 24
     */
    defaultRows?: number;
};

export type UseTerminalReturn = {
    terminalRows: number;
    terminalColumns: number;
};

/**
 * Hook to manage terminal state, including row count and alternate buffer.
 */
export function useTerminal(options: UseTerminalOptions = {}): UseTerminalReturn {
    const { enableAlternateBuffer = true, defaultRows = 24 } = options;

    const [terminalRows, setTerminalRows] = useState(() => {
        return process.stdout?.rows || defaultRows;
    });

    const [terminalColumns, setTerminalColumns] = useState(() => {
        return process.stdout?.columns || 80;
    });

    useEffect(() => {
        const updateDimensions = () => {
            if (process.stdout?.rows) {
                setTerminalRows(process.stdout.rows);
            }
            if (process.stdout?.columns) {
                setTerminalColumns(process.stdout.columns);
            }
        };

        // Update on resize
        process.stdout?.on('resize', updateDimensions);
        updateDimensions();

        return () => {
            process.stdout?.off('resize', updateDimensions);
        };
    }, []);

    // Activate alternate buffer (full-screen mode) and clear terminal on mount
    useEffect(() => {
        if (!enableAlternateBuffer || !process.stdout?.write) return;

        process.stdout.write(ANSI.ALTERNATE_BUFFER_ENTER);
        process.stdout.write(ANSI.BRACKETED_PASTE_ENABLE);
        process.stdout.write(ANSI.CLEAR_SCREEN + ANSI.CURSOR_HOME);

        return () => {
            process.stdout.write(ANSI.BRACKETED_PASTE_DISABLE);
            process.stdout.write(ANSI.ALTERNATE_BUFFER_EXIT);
        };
    }, [enableAlternateBuffer]);

    return { terminalRows, terminalColumns };
}
