export const UI_CONFIG = {
    // Layout settings
    LAYOUT: {
        STATUS_BAR_HEIGHT: 1,
    },
    // Turn settings
    TURN: {
        HEADER_COLOR_USER: "gray",
        HEADER_COLOR_ASSISTANT: "green",
        SUMMARY_PREFIX: "📝 ",
    },
    // Conversation display
    CONVERSATION: {
        // Tools to hide from timeline (internal/noisy tools)
        HIDDEN_TOOLS: ["think"],
        // Prefixes for compact conversation display
        PREFIXES: {
            USER: ">",
            ASSISTANT: "$",
        },
        // Whether to show reasoning steps (think, internal reasoning)
        SHOW_REASONING: false,
    },
    // Tool Component Layouts (Height in lines)
    TOOLS: {
        COMMON: {
            MARGIN_Y: 1, // Spacing between steps
        },
        RUN_CMD: {
            HEADER_HEIGHT: 4, // Header box (Border + Content + Padding + Margin)
            BORDER_HEIGHT: 2, // Top/Bottom borders
            MARGIN_INNER: 1,  // Padding
            FOOTER_HEIGHT: 4, // Exit code box (Margin + Border + Padding + Content)
            MAX_OUTPUT_LINES: 20,
        },
        THINK: {
            HEADER_HEIGHT: 1,
            BORDER_HEIGHT: 2,
            MARGIN_INNER: 1,
        },
        READ_FILE: {
            HEADER_HEIGHT: 4, // Header box (Border + Content + Padding + Margin)
            BORDER_HEIGHT: 2,
            MARGIN_INNER: 1,
            MAX_CONTENT_LINES: 30,
        },
        RESPOND: {
            HEADER_HEIGHT: 0,
            BORDER_HEIGHT: 0,
            MARGIN_INNER: 0,
        },
        CONTEXT: {
            HEIGHT: 1,
        },
    },
};
