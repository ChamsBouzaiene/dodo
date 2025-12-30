import React, { useState, useEffect, useRef } from "react";
import { Text, Box, useInput } from "ink";

/**
 * Props for the Input component.
 */
export type InputProps = {
    /** Current value of the input */
    value: string;
    /** Callback when value changes */
    onChange: (value: string) => void;
    /** Callback when Enter is pressed */
    onSubmit: (value: string) => void;
    /** Placeholder text when empty */
    placeholder?: string;
    /** Whether input is disabled */
    isDisabled?: boolean;
    /** Callback for Up arrow (history navigation) */
    onHistoryUp?: () => void;
    /** Callback for Down arrow (history navigation) */
    onHistoryDown?: () => void;
};

export const Input: React.FC<InputProps> = ({
    value,
    onChange,
    onSubmit,
    placeholder = "",
    isDisabled = false,
    onHistoryUp,
    onHistoryDown,
}) => {
    // Track cursor position locally. 
    const [cursorPos, setCursorPos] = useState(value.length);

    // Keep cursor valid if value length changes externally
    useEffect(() => {
        if (cursorPos > value.length) {
            setCursorPos(value.length);
        }
    }, [value, cursorPos]);

    // Use a ref to access latest state without re-creating the handler
    const stateRef = useRef({ value, cursorPos, onChange, onSubmit, isDisabled, onHistoryUp, onHistoryDown });
    stateRef.current = { value, cursorPos, onChange, onSubmit, isDisabled, onHistoryUp, onHistoryDown };

    useInput((input, key) => {
        const { value, cursorPos, onChange, onSubmit, isDisabled, onHistoryUp, onHistoryDown } = stateRef.current;

        if (isDisabled) return;

        // Handle return/enter
        if (key.return) {
            // Check for Shift+Enter. 
            if (key.shift) {
                const newValue = value.slice(0, cursorPos) + '\n' + value.slice(cursorPos);
                onChange(newValue);
                setCursorPos(p => p + 1);
                return;
            }
            onSubmit(value);
            return;
        }

        // Navigation
        if (key.leftArrow) {
            setCursorPos(p => Math.max(0, p - 1));
            return;
        }
        if (key.rightArrow) {
            setCursorPos(p => Math.min(value.length, p + 1));
            return;
        }

        // History Navigation
        if (key.upArrow) {
            onHistoryUp?.();
            return;
        }
        if (key.downArrow) {
            onHistoryDown?.();
            return;
        }

        // Backspace (delete before cursor)
        // Explicitly handle \x7f and \x08 for better compatibility.
        // On many MacBook terminals, the "Delete" key is identified as key.delete 
        // but physically sits where Backspace is expected.
        if (key.backspace || key.delete || input === '\x7f' || input === '\x08' || (key.ctrl && input === 'h')) {
            if (cursorPos > 0) {
                const newValue = value.slice(0, cursorPos - 1) + value.slice(cursorPos);
                onChange(newValue);
                setCursorPos(p => Math.max(0, p - 1));
            }
            return;
        }

        // Home/End
        if (input === '\u001b[H' || input === '\u001bOH') { // Home
            setCursorPos(0);
            return;
        }
        if (input === '\u001b[F' || input === '\u001bOF') { // End
            setCursorPos(value.length);
            return;
        }

        // Forward Delete (delete at cursor)
        // Usually triggered by Fn+Delete on Mac or Ctrl+D
        if (input === '\x1b[3~' || (key.ctrl && input === 'd')) {
            if (cursorPos < value.length) {
                const newValue = value.slice(0, cursorPos) + value.slice(cursorPos + 1);
                onChange(newValue);
            }
            return;
        }

        // Typing characters
        if (input && !key.ctrl && !key.meta) {
            // Handle multiline paste or special sequences
            if (input.includes('\n') || input.includes('\r')) {
                const normalizedInput = input.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
                const newValue = value.slice(0, cursorPos) + normalizedInput + value.slice(cursorPos);
                onChange(newValue);
                setCursorPos(p => p + normalizedInput.length);
                return;
            }

            // Normal typing (filter out ANSI escapes and control characters)
            // Only allow printable characters (32-126) or Tab (9)
            const charCode = input.charCodeAt(0);
            if (input.length === 1 && (
                (charCode >= 32 && charCode <= 126) || // Printable ASCII
                charCode === 9 || // Tab
                charCode > 127 // Extended ASCII/Unicode
            )) {
                const newValue = value.slice(0, cursorPos) + input + value.slice(cursorPos);
                onChange(newValue);
                setCursorPos(p => p + 1);
            }
        }
    }, { isActive: !isDisabled });

    // If empty and placeholder exists
    if (!value && placeholder) {
        return <Text color="gray">{placeholder}</Text>;
    }

    const lines = value.split('\n');
    let currentPos = 0;

    return (
        <Box flexDirection="column">
            {lines.map((line, lineIdx) => {
                const isLastLine = lineIdx === lines.length - 1;
                const lineStart = currentPos;
                const lineEnd = lineStart + line.length;

                const element = (
                    <Text key={lineIdx}>
                        {cursorPos >= lineStart && cursorPos <= lineEnd ? (
                            <>
                                <Text color="green">{line.slice(0, cursorPos - lineStart)}</Text>
                                <Text inverse color="green">{value[cursorPos] === '\n' ? '↵' : (line[cursorPos - lineStart] || " ")}</Text>
                                <Text color="green">{line.slice(cursorPos - lineStart + 1)}</Text>
                            </>
                        ) : (
                            <Text color="green">{line}</Text>
                        )}
                    </Text>
                );

                currentPos = lineEnd + 1; // +1 for the newline
                return element;
            })}
        </Box>
    );
};
