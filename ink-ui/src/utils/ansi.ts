/**
 * ANSI escape codes for terminal control
 */

export const ANSI = {
    // Screen / Buffer
    ALTERNATE_BUFFER_ENTER: '\x1b[?1049h',
    ALTERNATE_BUFFER_EXIT: '\x1b[?1049l',
    CLEAR_SCREEN: '\x1b[2J',
    CURSOR_HOME: '\x1b[H',

    // Mouse Tracking
    MOUSE_TRACKING_ENABLE: '\x1b[?1002h\x1b[?1006h',
    MOUSE_TRACKING_DISABLE: '\x1b[?1006l\x1b[?1002l',

    // Bracketed Paste
    BRACKETED_PASTE_ENABLE: '\x1b[?2004h',
    BRACKETED_PASTE_DISABLE: '\x1b[?2004l',
} as const;
