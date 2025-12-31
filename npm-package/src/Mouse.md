# Mouse Sequence Regex Strategy

This guide documents how Gemini CLI’s terminal UI detects and strips mouse sequences before they can corrupt keyboard input. It focuses on the regular expressions and helper functions defined in `packages/cli/src/ui/utils/input.ts` and reused throughout the mouse pipeline.

---

## 1. Mouse Protocols the CLI Understands

Terminals can report mouse activity in two legacy-compatible formats:

1. **SGR (1006) mode** – Emits escape sequences like `ESC [ < btn ; col ; row (m|M)` where the final letter indicates press/release.
2. **X10/X11 (1000) mode** – Emits `ESC [ M` followed by exactly three bytes encoding button + coordinates (offset by 32).

Gemini enables both formats by turning on `?1002h` (button-event tracking) and `?1006h` (SGR). To parse the stream reliably, the CLI keeps two regexes:

```ts
export const SGR_MOUSE_REGEX = /^\x1b\[<(\d+);(\d+);(\d+)([mM])/;
export const X11_MOUSE_REGEX = /^\x1b\[M([\s\S]{3})/;
```

Key takeaways:
- Both expressions are anchored (`^`) so matches only occur at the buffer head.
- Control characters appear literally (`\x1b` sequences) to avoid double escaping.
- Capturing groups expose button code, column, row, and press/release state (for SGR) or the raw 3-byte payload (for X11).

---

## 2. Prefix Detection Helpers

While parsing stdin, bytes often arrive chunked. The CLI uses lightweight prefix checks to decide when to keep buffering:

```ts
export function couldBeSGRMouseSequence(buffer: string): boolean {
  if (buffer.length === 0) return true;
  if (SGR_EVENT_PREFIX.startsWith(buffer)) return true;
  if (buffer.startsWith(SGR_EVENT_PREFIX)) return true;
  return false;
}
```

`SGR_EVENT_PREFIX` is `ESC [ <`, so this function returns `true` both for:
- Partial prefixes (`"\x1b"`, `"\x1b["`, …)
- Any already-started sequence that *might* continue (buffer begins with `ESC[<`)

`couldBeMouseSequence` extends the same logic to both SGR and X11 prefixes. These helpers are used in:

- **MouseProvider** – to avoid discarding partial sequences while buffering
- **`isIncompleteMouseSequence`** (see `mouse.ts`) – to decide when to wait for more bytes versus flushing garbage

---

## 3. Detecting Complete Sequences

`getMouseSequenceLength(buffer)` applies the regexes to the buffer head and returns the matched length. Anything non-zero means a full sequence is available and can be sliced off safely.

```ts
const len = getMouseSequenceLength(mouseBuffer);
if (len > 0) {
  const sequence = mouseBuffer.slice(0, len);
  // parse + consume
}
```

This allows the parser to differentiate between valid mouse input and other terminal traffic (keyboard keys, control codes, etc.).

---

## 4. Parsing into Structured Events

Once the regex matches, the CLI converts captures into usable metadata:

### SGR
```ts
const match = buffer.match(SGR_MOUSE_REGEX);
const buttonCode = parseInt(match[1], 10);
const col = parseInt(match[2], 10);
const row = parseInt(match[3], 10);
const action = match[4]; // 'm' release, 'M' press
```

From the `buttonCode`, bit masks determine:
- Scroll vs. click (`(buttonCode & 64) !== 0`)
- Movement flags (`buttonCode & 32`)
- Modifier keys (bits 2–4)

### X11
```ts
const payload = match[1];
const buttonByte = payload.charCodeAt(0) - 32;
const col = payload.charCodeAt(1) - 32;
const row = payload.charCodeAt(2) - 32;
```

This format lacks explicit release notifications and modifiers, so the parser makes best-effort guesses (e.g., treat button code `3` as “release”).

Both branches return a unified `MouseEvent`:
```ts
interface MouseEvent {
  name: MouseEventName; // 'scroll-up', 'left-press', etc.
  col: number;
  row: number;
  shift: boolean;
  meta: boolean;
  ctrl: boolean;
  button: 'left' | 'middle' | 'right' | 'none';
}
```

---

## 5. Handling Incomplete or Garbled Input

`isIncompleteMouseSequence(buffer)` leverages the same regex and prefix helpers to tell whether the current buffer:
1. Contains a full event → return `false` (parsers will consume it)
2. Looks like a prefix of an event → return `true` (keep buffering)
3. Looks like garbage → return `false`, prompting the caller to discard bytes until the next escape character

This mechanism keeps the parser resilient to partial writes, copy/paste, and terminals that interleave mouse and keyboard data.

---

## 6. Where These Functions Are Used

| Location | Purpose |
| --- | --- |
| `MouseProvider` (`MouseContext.tsx`) | Reads stdin, accumulates bytes, and repeatedly runs the regex pipeline. |
| `nonKeyboardEventFilter` (`KeypressContext.tsx`) | Uses `parseMouseEvent` (which uses the same regexes) to drop mouse sequences before they reach text input handlers. |
| `ScrollProvider` (`ScrollProvider.tsx`) | Receives structured `MouseEvent`s; doesn’t touch raw regexes directly but relies on the parser to classify scroll events. |

Because both keyboard and mouse systems rely on the same parser, there is no risk that a scroll sequence slips past the filters—any future adjustments (e.g., supporting new protocols) only require updating the shared regex definitions and parser logic.

---

## 7. Extending or Reusing the Strategy

1. **Need to support another protocol?**  
   Add a new prefix constant + regex, update `couldBeMouseSequence`, `getMouseSequenceLength`, and `parseMouseEvent` to recognize it.

2. **Want to log unrecognized sequences?**  
   Hook into the `else` branch inside `MouseProvider.handleData` where garbage bytes are dropped.

3. **Porting to another framework?**  
   The regex logic is framework-agnostic. You only need access to the raw stdin stream and a place to buffer bytes between reads.

4. **Guarding keyboard input elsewhere**  
   Reuse `parseMouseEvent` (or the regexes directly) to filter out mouse sequences inside any text input component that shares stdin with the mouse listener.

With these building blocks, you can reliably detect, parse, and suppress mouse scroll sequences in any agentic CLI environment.

