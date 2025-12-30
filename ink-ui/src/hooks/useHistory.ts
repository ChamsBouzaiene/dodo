import { useState, useCallback } from "react";

export type UseHistoryReturn = {
    history: string[];
    historyIndex: number;
    addToHistory: (command: string) => void;
    navigateHistory: (direction: "up" | "down") => { index: number; value: string | undefined };
    resetHistoryIndex: () => void;
};

export function useHistory(): UseHistoryReturn {
    const [history, setHistory] = useState<string[]>([]);
    const [historyIndex, setHistoryIndex] = useState(-1);

    const addToHistory = useCallback((command: string) => {
        setHistory((prev) => {
            // Don't add duplicates consecutively
            const last = prev[prev.length - 1];
            if (last !== command) {
                return [...prev, command];
            }
            return prev;
        });
        setHistoryIndex(-1); // Always reset index on new entry
    }, []);

    const resetHistoryIndex = useCallback(() => setHistoryIndex(-1), []);

    const navigateHistory = useCallback(
        (direction: "up" | "down") => {
            let newIndex = 0;
            let returnValue: string | undefined = undefined;

            setHistoryIndex((prev) => {
                newIndex = prev;
                if (direction === "up") {
                    newIndex = Math.min(prev + 1, history.length - 1);
                } else {
                    newIndex = Math.max(prev - 1, -1);
                }
                return newIndex;
            });

            // We need to calculate the value based on the *calculated* newIndex,
            // but state updates are async, so we use the logic directly here.
            // Wait, the setHistoryIndex callback style above updates state but for return value
            // we need to replicate the logic because we can't await state.
            // Actually, cleaner logic: calculate index first.

            // Re-doing logic to clear up async confusion:
            // The caller needs the new string immediately to update input.
            // So we will return the calculation function.
        },
        [history]
    );

    // Rewriting navigateHistory to be synchronous-friendly for the caller
    // The caller (App.tsx) keeps the input state, but we track the index state.
    // We expose a function that accepts current index (or we assume state is up to date?)
    // Actually, standard React pattern: we return the value derived from state.
    // But wait, "navigate" is an action.

    // Let's use a slightly different pattern for robust navigation:
    // "moveUp()" sets state and returns value? No, return value lags.

    // Alternative: The hook returns `currentHistoryValue`.
    // If index === -1, value is undefined (or null).
    // When user types, they override.

    // But standard CLI history:
    // Typed: "abc" -> Up -> "prev command" -> Down -> "abc" (restored).
    // Complexity: We need to store the "draft" input when moving up.
    // That's too much state for this simple hook.

    // Let's stick to the App.tsx implementation logic but encapsulated.
    // We return a specialized `navigate` function that updates index state AND returns the new string.

    const navigate = useCallback((direction: "up" | "down", currentHistoryIndex: number, currentHistory: string[]) => {
        let newIndex = currentHistoryIndex;
        if (direction === "up") {
            newIndex = Math.min(currentHistoryIndex + 1, currentHistory.length - 1);
        } else {
            newIndex = Math.max(currentHistoryIndex - 1, -1);
        }

        let newValue: string | undefined = undefined;
        if (newIndex >= 0 && newIndex < currentHistory.length) {
            // history is oldest -> newest
            // index 0 = most recent (standard convention for up arrow)
            // Wait, App.tsx used: index 0 = 1 back?
            // Let's check App.tsx logic:
            // newIndex = prev + 1. (0 -> 1).
            // value = history[history.length - 1 - newIndex]
            // If length 3 [a, b, c]. index 0. value = 3 - 1 - 0 = 2 -> 'c'.
            // Correct via App.tsx logic.
            newValue = currentHistory[currentHistory.length - 1 - newIndex];
        }

        return { index: newIndex, value: newValue };
    }, []);

    return {
        history,
        historyIndex,
        addToHistory,
        navigateHistory: (direction) => {
            const res = navigate(direction, historyIndex, history);
            setHistoryIndex(res.index);
            return res;
        },
        resetHistoryIndex
    };
}
