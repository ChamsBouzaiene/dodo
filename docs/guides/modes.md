# Operating Modes

Dodo isn't one-size-fits-all. Different tasks require different "mindsets." Dodo supports specialized **Modes** that change the system prompt and available tools.

Switch modes anytime using the `/mode` command.

## 1. Coder Mode (Default)

**Goal**: Implementation & Fixes.

This is the standard day-to-day mode. The agent assumes it is a Senior Software Engineer paired with you. 

-   **Strengths**: Writing code, refactoring, fixing bugs, running tests.
-   **Tools**: Full access (Read, Write, Execute).
-   **Behavior**: Biased towards action. It will try to solve the problem by editing files.

## 2. Architect Mode

**Goal**: High-Level Design & Planning.

Use this mode when you are starting a complex feature and want to discuss the approach *before* writing code.

-   **Strengths**: System design, directory structure planning, technology choices.
-   **Tools**: Read-only (mostly). It can read files to understand the current state but is discouraged from making edits.
-   **Behavior**: Biased towards discussion and creating `implementation_plan.md` artifacts.

## 3. Ask Mode

**Goal**: Q&A and Exploration.

Use this mode when you just want to understand the codebase without any risk of changing it.

-   **Strengths**: "Where is the auth logic defined?", "How does the billing system work?".
-   **Tools**: Read-only (Search, Read). Write tools are disabled.
-   **Behavior**: Purely conversational. It acts as an expert guide to your repository.
