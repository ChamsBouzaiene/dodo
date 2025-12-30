import { useState, useCallback, useRef, useEffect } from "react";
import type { EngineClient } from "../engineClient.js";
import { useEngineEvents } from "./useEngineEvents.js";

export function useResponseStream(
    client: EngineClient,
    appendAssistantContent: (content: string, replace: boolean) => void,
    markLastTurnDone: () => void
) {
    const [currentThought, setCurrentThought] = useState("");

    const bufferRef = useRef("");
    const timerRef = useRef<NodeJS.Timeout | null>(null);
    const lastUpdateRef = useRef(0);
    const THROTTLE_MS = 50; // 20fps cap for text updates

    const flushBuffer = useCallback((final: boolean) => {
        if (timerRef.current) {
            clearTimeout(timerRef.current);
            timerRef.current = null;
        }

        const textToFlush = bufferRef.current;
        bufferRef.current = "";
        lastUpdateRef.current = Date.now();

        if (textToFlush) {
            setCurrentThought((prev) => prev + textToFlush);
            appendAssistantContent(textToFlush, false);
        }

        if (final) {
            // Clear currentThought when turn completes to stop spinner
            setCurrentThought("");
            markLastTurnDone();
        }
    }, [appendAssistantContent, markLastTurnDone]);

    useEffect(() => {
        return () => {
            if (timerRef.current) clearTimeout(timerRef.current);
        };
    }, []);

    const handleAssistantText = useCallback(
        (content: string, replace: boolean, final: boolean) => {
            if (replace) {
                if (timerRef.current) clearTimeout(timerRef.current);
                timerRef.current = null;
                bufferRef.current = "";
                lastUpdateRef.current = Date.now();
                setCurrentThought(final ? "" : content);
                appendAssistantContent(content, true);
                if (final) markLastTurnDone();
                return;
            }

            bufferRef.current += content;
            const now = Date.now();

            if (final || now - lastUpdateRef.current > THROTTLE_MS) {
                flushBuffer(final);
            } else if (!timerRef.current) {
                timerRef.current = setTimeout(() => flushBuffer(false), THROTTLE_MS);
            }
        },
        [flushBuffer, appendAssistantContent, markLastTurnDone]
    );

    useEngineEvents(client, {
        onAssistantText: handleAssistantText,
    });

    return {
        currentThought,
        setCurrentThought
    };
}
