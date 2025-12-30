# Ink UI Performance & Best Practices

This document summarizes critical learnings from debugging high-performance CLI applications built with `ink`.

## The "Event Loop Starvation" Problem

### Symptoms
*   Application appears frozen during high-frequency updates (e.g., streaming text).
*   `Ctrl+C` and other keypresses are ignored or delayed significantly.
*   The UI might still be updating, but the process is unresponsive to input.

### Root Cause
React's standard state update pattern (`setState(prev => [...prev, newItem])`) is **immutable**. When used with high-frequency data streams (like character-by-character LLM output):
1.  **O(N) Copying**: Every update copies the entire state array/object.
2.  **GC Pressure**: Thousands of temporary objects are created and discarded per second.
3.  **Thread Blocking**: The combination of array copying and Garbage Collection blocks the Node.js Event Loop.
4.  **Input Starvation**: `stdin` data events (keypresses) are queued but not processed because the main thread is too busy.

### Solution: Mutable Refs + Throttled Rendering
Decouple the **Data State** from the **UI State**.

1.  **Use `useRef` for Data**: Store the authoritative state in a mutable `useRef`.
    ```typescript
    const turnsRef = useRef<Turn[]>([]);
    ```
2.  **Mutate in Place**: Update the ref directly without copying.
    ```typescript
    turnsRef.current.push(newTurn); // O(1)
    ```
3.  **Throttle UI Updates**: Only sync the Ref to React State at a human-perceivable framerate (e.g., 10-20fps).
    ```typescript
    const throttledUpdate = useCallback(() => {
      const now = Date.now();
      if (now - lastRender.current > 100) { // 100ms = 10fps
        setTurns([...turnsRef.current]); // Shallow copy for React
        lastRender.current = now;
      }
    }, []);
    ```

## The "Scroll Freeze" Problem

### Symptoms
*   Scrolling (especially with trackpads) causes the application to hang.
*   CPU usage spikes during scrolling.

### Root Cause
Trackpads generate scroll events at a very high rate (hundreds per second). If the scroll handler triggers a re-render or layout calculation (`measureElement`) for every event, the renderer falls behind, causing a backlog of work that freezes the UI.

### Solution: Event Throttling
Throttle input handlers to a reasonable framerate (e.g., 30fps).

```typescript
const handleScroll = useCallback((delta: number) => {
  const now = Date.now();
  if (now - lastScrollTime.current < 33) return; // Limit to ~30fps
  lastScrollTime.current = now;
  
  // ... perform scroll logic ...
}, []);
```

## General Best Practices for Ink

1.  **Avoid `measureElement` in loops**: Layout measurement is expensive. Cache heights where possible or use fixed heights for large lists.
2.  **Isolate Heavy Components**: If a component updates frequently, try to prevent it from causing re-renders in parent or sibling components.
3.  **Monitor the Event Loop**: If `Ctrl+C` stops working, your Event Loop is blocked. Check for tight loops or excessive synchronous work.
4.  **Use `process.kill(process.pid, 'SIGKILL')` for emergency exits**: If `ink`'s unmount process hangs, a hard kill might be necessary during debugging.
