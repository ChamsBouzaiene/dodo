/**
 * Shared mouse utilities and types
 */

// Mouse event types
export type MouseEventType = "scroll-up" | "scroll-down" | "click" | "move" | "release";

export type MouseEvent = {
    type: MouseEventType;
    x: number;
    y: number;
    button: number;
};

// SGR mouse sequence regex
export const SGR_REGEX = /\x1b\[<(\d+);(\d+);(\d+)([Mm])/g;

export type MouseHandler = (event: MouseEvent) => void;
