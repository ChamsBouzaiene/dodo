# Complete Implementation Guide: Virtualized List with Ink

This guide walks you through implementing a virtualized list from scratch, step by step. You'll learn how to estimate heights, measure actual heights with Ink, and render only visible items.

---

## Table of Contents

1. [Understanding the Problem](#understanding-the-problem)
2. [Step 1: Basic Setup](#step-1-basic-setup)
3. [Step 2: Height Estimation System](#step-2-height-estimation-system)
4. [Step 3: Measuring Heights with Ink](#step-3-measuring-heights-with-ink)
5. [Step 4: Calculating Cumulative Offsets](#step-4-calculating-cumulative-offsets)
6. [Step 5: Determining Visible Range](#step-5-determining-visible-range)
7. [Step 6: Rendering Only Visible Items](#step-6-rendering-only-visible-items)
8. [Step 7: Using Spacer Boxes](#step-7-using-spacer-boxes)
9. [Step 8: Managing Scroll Position](#step-8-managing-scroll-position)
10. [Step 9: Complete Implementation](#step-9-complete-implementation)
11. [Step 10: Helper Utilities](#step-10-helper-utilities)
12. [Testing Your Implementation](#testing-your-implementation)

---

## Understanding the Problem

### The Challenge

When you have a large list (e.g., 1000 items), rendering all items at once:
- **Slows down rendering**: React has to create 1000+ components
- **Wastes memory**: All items exist in memory even if not visible
- **Causes lag**: Terminal rendering becomes slow

### The Solution: Virtualization

**Virtualization** means:
- Only render items that are **visible** (or about to be visible)
- Use **spacer boxes** to maintain correct scroll position
- **Measure actual heights** after rendering to correct estimates

### Example

```
Container height: 10 lines
Total items: 1000
Items visible: ~10-15 items

Instead of rendering 1000 items, render only 10-15!
```

---

## Step 1: Basic Setup

### 1.1 Install Dependencies

```bash
npm install ink react
```

### 1.2 Create Basic Component Structure

```tsx
import { useState, useRef, useLayoutEffect, useMemo } from 'react';
import { Box, type DOMElement, measureElement } from 'ink';

type VirtualizedListProps<T> = {
  data: T[];
  renderItem: (info: { item: T; index: number }) => React.ReactElement;
  estimatedItemHeight: (index: number) => number;
  keyExtractor: (item: T, index: number) => string;
};

function VirtualizedList<T>({
  data,
  renderItem,
  estimatedItemHeight,
  keyExtractor,
}: VirtualizedListProps<T>) {
  // We'll add state and logic here step by step
  
  return (
    <Box>
      {/* Items will be rendered here */}
    </Box>
  );
}
```

### 1.3 Set Up State Variables

```tsx
function VirtualizedList<T>({ data, renderItem, estimatedItemHeight, keyExtractor }: VirtualizedListProps<T>) {
  // Container reference (to measure viewport height)
  const containerRef = useRef<DOMElement>(null);
  const [containerHeight, setContainerHeight] = useState(0);

  // Item references (to measure each item's height)
  const itemRefs = useRef<Array<DOMElement | null>>([]);

  // Actual measured heights (starts empty, filled as items are measured)
  const [heights, setHeights] = useState<number[]>([]);

  // Scroll position (we'll implement this later)
  const [scrollTop, setScrollTop] = useState(0);

  return (
    <Box ref={containerRef}>
      {/* Items will be rendered here */}
    </Box>
  );
}
```

**Key Points**:
- `containerRef`: Reference to the scrollable container
- `itemRefs`: Array of references to each rendered item
- `heights`: Array storing actual measured heights `[height0, height1, ...]`
- `containerHeight`: Height of the visible viewport

---

## Step 2: Height Estimation System

### 2.1 Why We Need Estimates

Before we can render items, we need to know their heights to:
- Calculate which items are visible
- Calculate scroll position
- Calculate spacer heights

But we can't measure items until they're rendered! So we use **estimates** first.

### 2.2 Initialize Height Estimates

When new items are added to the data array, initialize their heights with estimates:

```tsx
useEffect(() => {
  setHeights((prevHeights) => {
    // If data length matches, no changes needed
    if (data.length === prevHeights.length) {
      return prevHeights;
    }

    const newHeights = [...prevHeights];

    // If data shrunk, trim heights array
    if (data.length < prevHeights.length) {
      newHeights.length = data.length;
      return newHeights;
    }

    // If data grew, add estimates for new items
    for (let i = prevHeights.length; i < data.length; i++) {
      newHeights[i] = estimatedItemHeight(i);
    }

    return newHeights;
  });
}, [data, estimatedItemHeight]);
```

**Example**:
```tsx
// Initial state: heights = []
// Data has 3 items
// After effect: heights = [5, 5, 5] (using estimate of 5 for each)

// Data grows to 5 items
// After effect: heights = [5, 5, 5, 5, 5] (added 2 more estimates)
```

### 2.3 Writing a Good `estimatedItemHeight` Function

```tsx
// Example 1: Fixed height (all items same size)
const estimatedItemHeight = () => 3; // 3 lines per item

// Example 2: Variable height based on item content
const estimatedItemHeight = (index: number) => {
  const item = data[index];
  if (item.type === 'header') return 5;
  if (item.type === 'message') return item.lines.length + 2;
  return 3;
};

// Example 3: Based on index (items get taller)
const estimatedItemHeight = (index: number) => {
  return 2 + Math.floor(index / 10); // Gradually increases
};
```

**Tips**:
- **Be conservative**: Slightly overestimate rather than underestimate
- **Consider content**: If items have variable content, estimate based on typical size
- **Start simple**: Fixed height is easiest to start with

---

## Step 3: Measuring Heights with Ink

### 3.1 Understanding `measureElement`

Ink's `measureElement` function returns the actual rendered dimensions of an element:

```tsx
import { measureElement, type DOMElement } from 'ink';

const element: DOMElement = /* your element ref */;
const measurement = measureElement(element);
// measurement = { width: 80, height: 5 }
```

### 3.2 Measure Container Height

First, measure the container (viewport) height:

```tsx
useLayoutEffect(() => {
  if (containerRef.current) {
    const height = Math.round(measureElement(containerRef.current).height);
    if (containerHeight !== height) {
      setContainerHeight(height);
    }
  }
});
```

**Why `useLayoutEffect`?**
- Runs **synchronously** after DOM mutations but before paint
- Ensures measurements happen after render but before display
- Runs on **every render** to catch size changes

**Why check `containerHeight !== height`?**
- Prevents infinite loops
- Only updates state when value actually changes

### 3.3 Measure Item Heights

Measure heights of currently visible items:

```tsx
useLayoutEffect(() => {
  // Measure container
  if (containerRef.current) {
    const height = Math.round(measureElement(containerRef.current).height);
    if (containerHeight !== height) {
      setContainerHeight(height);
    }
  }

  // Measure visible items (we'll calculate startIndex/endIndex in next step)
  let newHeights: number[] | null = null;
  
  for (let i = startIndex; i <= endIndex; i++) {
    const itemRef = itemRefs.current[i];
    if (itemRef) {
      const measuredHeight = Math.round(measureElement(itemRef).height);
      
      // Only update if height changed
      if (measuredHeight !== heights[i]) {
        // Lazy copy: only create new array when first change detected
        if (!newHeights) {
          newHeights = [...heights];
        }
        newHeights[i] = measuredHeight;
      }
    }
  }

  // Only update state if something changed
  if (newHeights) {
    setHeights(newHeights);
  }
});
```

**Key Points**:
- **Lazy copying**: Only create new array when first change detected
- **Conditional updates**: Only update state if heights actually changed
- **Round values**: `Math.round()` ensures integer heights (cleaner calculations)

### 3.4 The Measurement Flow

```
1. Component renders with estimated heights
2. Items are rendered to DOM
3. useLayoutEffect runs
4. measureElement() measures actual heights
5. Heights state updated if different
6. Component re-renders with new heights
7. Offsets recalculated (next step)
8. Visible range recalculated
9. Process repeats until heights stabilize
```

---

## Step 4: Calculating Cumulative Offsets

### 4.1 What Are Offsets?

**Offsets** are cumulative heights - they tell us where each item starts vertically.

```
Item 0: height = 3 → offset[0] = 0        (starts at 0)
Item 1: height = 5 → offset[1] = 3        (starts at 3)
Item 2: height = 2 → offset[2] = 8        (starts at 8)
Item 3: height = 4 → offset[3] = 10       (starts at 10)
Item 4: height = 6 → offset[4] = 14       (starts at 14)
                                    total = 20

offsets = [0, 3, 8, 10, 14, 20]
```

**Why we need this**:
- Quickly find which item contains a scroll position
- Calculate spacer heights
- Calculate total scrollable height

### 4.2 Calculate Offsets with useMemo

```tsx
const { totalHeight, offsets } = useMemo(() => {
  const offsets: number[] = [0]; // First offset is always 0
  let totalHeight = 0;

  for (let i = 0; i < data.length; i++) {
    // Use actual height if measured, otherwise use estimate
    const height = heights[i] ?? estimatedItemHeight(i);
    totalHeight += height;
    offsets.push(totalHeight);
  }

  return { totalHeight, offsets };
}, [heights, data, estimatedItemHeight]);
```

**How it works**:
1. Start with `[0]` (first item starts at position 0)
2. For each item, add its height to `totalHeight`
3. Push `totalHeight` to `offsets` (this is where the NEXT item starts)
4. Result: `offsets[i]` = sum of heights from item 0 to item i-1

**Example walkthrough**:
```tsx
// heights = [3, 5, 2, 4, 6]
// data.length = 5

i=0: height=3, totalHeight=3,  offsets=[0, 3]
i=1: height=5, totalHeight=8,  offsets=[0, 3, 8]
i=2: height=2, totalHeight=10, offsets=[0, 3, 8, 10]
i=3: height=4, totalHeight=14, offsets=[0, 3, 8, 10, 14]
i=4: height=6, totalHeight=20, offsets=[0, 3, 8, 10, 14, 20]

// Final: offsets = [0, 3, 8, 10, 14, 20], totalHeight = 20
```

**Why `useMemo`?**
- Offsets calculation is O(n) - expensive for large lists
- Only recalculate when `heights`, `data`, or `estimatedItemHeight` change
- Prevents unnecessary recalculations on every render

---

## Step 5: Determining Visible Range

### 5.1 The Problem

Given:
- `scrollTop`: How many pixels we've scrolled down
- `containerHeight`: Height of visible viewport
- `offsets`: Where each item starts

Find:
- `startIndex`: First item that's (partially) visible
- `endIndex`: Last item that's (partially) visible

### 5.2 Finding startIndex

We need the **last** offset that's **less than or equal** to `scrollTop`:

```tsx
function findLastIndex<T>(
  array: T[],
  predicate: (value: T, index: number, obj: T[]) => unknown,
): number {
  for (let i = array.length - 1; i >= 0; i--) {
    if (predicate(array[i]!, i, array)) {
      return i;
    }
  }
  return -1;
}

// Find the last offset <= scrollTop
const lastOffsetIndex = findLastIndex(offsets, (offset) => offset <= scrollTop);

// The item at that index contains scrollTop
// But we want the item BEFORE that (since offsets[i] is where item i starts)
const startIndex = Math.max(0, lastOffsetIndex - 1);
```

**Example**:
```tsx
offsets = [0, 3, 8, 10, 14, 20]
scrollTop = 5

// Find last offset <= 5
// offset[0] = 0 <= 5 ✓
// offset[1] = 3 <= 5 ✓
// offset[2] = 8 > 5 ✗
// So lastOffsetIndex = 1

// startIndex = 1 - 1 = 0
// But wait, item 0 ends at offset 3, and we're at scrollTop 5
// So we should be in item 1!

// Actually, we need to check: if scrollTop is past offset[i], we're in item i
// So: find last offset <= scrollTop, that's the item we're in
// startIndex = lastOffsetIndex (not lastOffsetIndex - 1)

// Correction:
const startIndex = Math.max(0, findLastIndex(offsets, (offset) => offset <= scrollTop));
```

**Wait, let's think more carefully**:
- `offsets[0] = 0` means item 0 starts at 0
- `offsets[1] = 3` means item 1 starts at 3
- If `scrollTop = 5`, we're past item 0 (which ends at 3) and in item 1
- So `startIndex` should be the index where `offsets[i] <= scrollTop < offsets[i+1]`
- That's exactly `findLastIndex(offsets, offset => offset <= scrollTop)`

But we also want a small buffer (render items slightly above viewport):

```tsx
const startIndex = Math.max(
  0,
  findLastIndex(offsets, (offset) => offset <= scrollTop) - 1
);
```

The `-1` gives us one item buffer above the viewport.

### 5.3 Finding endIndex

Find the first offset that's **greater than** `scrollTop + containerHeight`:

```tsx
const endIndexOffset = offsets.findIndex(
  (offset) => offset > scrollTop + containerHeight
);

const endIndex =
  endIndexOffset === -1
    ? data.length - 1  // All items visible
    : Math.min(data.length - 1, endIndexOffset);
```

**Example**:
```tsx
offsets = [0, 3, 8, 10, 14, 20]
scrollTop = 5
containerHeight = 10

// Find first offset > (5 + 10) = 15
// offset[0] = 0 <= 15 ✗
// offset[1] = 3 <= 15 ✗
// offset[2] = 8 <= 15 ✗
// offset[3] = 10 <= 15 ✗
// offset[4] = 14 <= 15 ✗
// offset[5] = 20 > 15 ✓

// endIndexOffset = 5
// endIndex = min(4, 5) = 4 (last item index)
```

### 5.4 Complete Visible Range Calculation

```tsx
const startIndex = Math.max(
  0,
  findLastIndex(offsets, (offset) => offset <= scrollTop) - 1
);

const endIndexOffset = offsets.findIndex(
  (offset) => offset > scrollTop + containerHeight
);

const endIndex =
  endIndexOffset === -1
    ? data.length - 1
    : Math.min(data.length - 1, endIndexOffset);
```

**Visual Example**:
```
Total height: 20
Container height: 10
Scroll position: 5

offsets: [0, 3, 8, 10, 14, 20]
         |  |  |   |   |   |
         |  |  |   |   |   └─ Item 4 ends
         |  |  |   |   └─ Item 4 starts
         |  |  |   └─ Item 3 ends
         |  |  └─ Item 3 starts
         |  └─ Item 2 ends
         └─ Item 2 starts

Viewport (scrollTop=5 to scrollTop=15):
         [==========]
              |  |
         Item 1, 2, 3 are visible

startIndex = 1 (with -1 buffer, so 0)
endIndex = 3
```

---

## Step 6: Rendering Only Visible Items

### 6.1 Create Item References

We need references to each rendered item so we can measure them:

```tsx
const itemRefs = useRef<Array<DOMElement | null>>([]);

// When rendering, store refs
const renderedItems = [];
for (let i = startIndex; i <= endIndex; i++) {
  const item = data[i];
  if (item) {
    renderedItems.push(
      <Box
        key={keyExtractor(item, i)}
        width="100%"
        ref={(el) => {
          itemRefs.current[i] = el;
        }}
      >
        {renderItem({ item, index: i })}
      </Box>
    );
  }
}
```

**Key Points**:
- `keyExtractor`: Provides unique key for React reconciliation
- `ref` callback: Stores element reference in `itemRefs.current[i]`
- Only items in `[startIndex, endIndex]` range are rendered

### 6.2 Complete Rendering Logic

```tsx
const renderedItems = [];
for (let i = startIndex; i <= endIndex; i++) {
  const item = data[i];
  if (item) {
    renderedItems.push(
      <Box
        key={keyExtractor(item, i)}
        width="100%"
        ref={(el) => {
          itemRefs.current[i] = el;
        }}
      >
        {renderItem({ item, index: i })}
      </Box>
    );
  }
}
```

**Why `width="100%"`?**
- Ensures items take full container width
- Important for correct height measurements

---

## Step 7: Using Spacer Boxes

### 7.1 Why Spacers Are Needed

If we only render visible items, the total height would be wrong:

```
Without spacers:
- Container height: 10
- Visible items: 3 items, total height 8
- Scrollbar thinks total height is 8 (wrong!)
- Can't scroll to items below

With spacers:
- Top spacer: height of items 0 to startIndex-1
- Visible items: startIndex to endIndex
- Bottom spacer: height of items endIndex+1 to end
- Total height: correct!
```

### 7.2 Calculate Spacer Heights

```tsx
// Top spacer: sum of heights before startIndex
const topSpacerHeight = offsets[startIndex] ?? 0;

// Bottom spacer: total height minus height up to endIndex+1
const bottomSpacerHeight = totalHeight - (offsets[endIndex + 1] ?? totalHeight);
```

**Example**:
```tsx
offsets = [0, 3, 8, 10, 14, 20]
startIndex = 1
endIndex = 3
totalHeight = 20

topSpacerHeight = offsets[1] = 3 (height of item 0)
bottomSpacerHeight = 20 - offsets[4] = 20 - 14 = 6 (height of item 4)

// Items rendered: 1, 2, 3 (heights: 5, 2, 4 = 11)
// Total: 3 + 11 + 6 = 20 ✓
```

### 7.3 Render with Spacers

```tsx
return (
  <Box
    ref={containerRef}
    overflowY="scroll"
    overflowX="hidden"
    scrollTop={scrollTop}
    width="100%"
    height="100%"
    flexDirection="column"
  >
    <Box flexShrink={0} width="100%" flexDirection="column">
      {/* Top spacer */}
      <Box height={topSpacerHeight} flexShrink={0} />
      
      {/* Visible items */}
      {renderedItems}
      
      {/* Bottom spacer */}
      <Box height={bottomSpacerHeight} flexShrink={0} />
    </Box>
  </Box>
);
```

**Key Points**:
- `flexShrink={0}`: Prevents spacers from shrinking
- `overflowY="scroll"`: Enables scrolling
- `scrollTop={scrollTop}`: Controls scroll position

---

## Step 8: Managing Scroll Position

### 8.1 Scroll Anchor System

Instead of storing raw `scrollTop`, we use a "scroll anchor" that's more stable:

```tsx
type ScrollAnchor = {
  index: number;  // Which item is at top of viewport
  offset: number; // How many pixels into that item
};

const [scrollAnchor, setScrollAnchor] = useState<ScrollAnchor>({ index: 0, offset: 0 });
```

**Why this is better**:
- Stable when data changes
- Self-correcting when heights are measured
- Prevents jumps

### 8.2 Convert Anchor to scrollTop

```tsx
const scrollTop = useMemo(() => {
  const offset = offsets[scrollAnchor.index];
  if (typeof offset !== 'number') {
    return 0;
  }
  return offset + scrollAnchor.offset;
}, [scrollAnchor, offsets]);
```

### 8.3 Convert scrollTop to Anchor

```tsx
function getAnchorForScrollTop(
  scrollTop: number,
  offsets: number[],
): ScrollAnchor {
  const index = findLastIndex(offsets, (offset) => offset <= scrollTop);
  if (index === -1) {
    return { index: 0, offset: 0 };
  }
  return { index, offset: scrollTop - offsets[index]! };
}
```

### 8.4 Handle Scroll Events

```tsx
const handleScroll = (newScrollTop: number) => {
  const clampedScrollTop = Math.max(
    0,
    Math.min(totalHeight - containerHeight, newScrollTop)
  );
  setScrollAnchor(getAnchorForScrollTop(clampedScrollTop, offsets));
};
```

---

## Step 9: Complete Implementation

Here's the complete code with all pieces together:

```tsx
import {
  useState,
  useRef,
  useLayoutEffect,
  useEffect,
  useMemo,
  useCallback,
} from 'react';
import { Box, type DOMElement, measureElement } from 'ink';

type VirtualizedListProps<T> = {
  data: T[];
  renderItem: (info: { item: T; index: number }) => React.ReactElement;
  estimatedItemHeight: (index: number) => number;
  keyExtractor: (item: T, index: number) => string;
};

type ScrollAnchor = {
  index: number;
  offset: number;
};

function findLastIndex<T>(
  array: T[],
  predicate: (value: T, index: number, obj: T[]) => unknown,
): number {
  for (let i = array.length - 1; i >= 0; i--) {
    if (predicate(array[i]!, i, array)) {
      return i;
    }
  }
  return -1;
}

function VirtualizedList<T>({
  data,
  renderItem,
  estimatedItemHeight,
  keyExtractor,
}: VirtualizedListProps<T>) {
  // Refs
  const containerRef = useRef<DOMElement>(null);
  const itemRefs = useRef<Array<DOMElement | null>>([]);

  // State
  const [containerHeight, setContainerHeight] = useState(0);
  const [heights, setHeights] = useState<number[]>([]);
  const [scrollAnchor, setScrollAnchor] = useState<ScrollAnchor>({
    index: 0,
    offset: 0,
  });

  // Initialize height estimates when data changes
  useEffect(() => {
    setHeights((prevHeights) => {
      if (data.length === prevHeights.length) {
        return prevHeights;
      }

      const newHeights = [...prevHeights];
      if (data.length < prevHeights.length) {
        newHeights.length = data.length;
      } else {
        for (let i = prevHeights.length; i < data.length; i++) {
          newHeights[i] = estimatedItemHeight(i);
        }
      }
      return newHeights;
    });
  }, [data, estimatedItemHeight]);

  // Calculate offsets
  const { totalHeight, offsets } = useMemo(() => {
    const offsets: number[] = [0];
    let totalHeight = 0;
    for (let i = 0; i < data.length; i++) {
      const height = heights[i] ?? estimatedItemHeight(i);
      totalHeight += height;
      offsets.push(totalHeight);
    }
    return { totalHeight, offsets };
  }, [heights, data, estimatedItemHeight]);

  // Convert anchor to scrollTop
  const scrollTop = useMemo(() => {
    const offset = offsets[scrollAnchor.index];
    if (typeof offset !== 'number') {
      return 0;
    }
    return offset + scrollAnchor.offset;
  }, [scrollAnchor, offsets]);

  // Get anchor from scrollTop
  const getAnchorForScrollTop = useCallback(
    (scrollTop: number, offsets: number[]): ScrollAnchor => {
      const index = findLastIndex(offsets, (offset) => offset <= scrollTop);
      if (index === -1) {
        return { index: 0, offset: 0 };
      }
      return { index, offset: scrollTop - offsets[index]! };
    },
    [],
  );

  // Determine visible range
  const startIndex = Math.max(
    0,
    findLastIndex(offsets, (offset) => offset <= scrollTop) - 1,
  );

  const endIndexOffset = offsets.findIndex(
    (offset) => offset > scrollTop + containerHeight,
  );

  const endIndex =
    endIndexOffset === -1
      ? data.length - 1
      : Math.min(data.length - 1, endIndexOffset);

  // Measure heights
  useLayoutEffect(() => {
    // Measure container
    if (containerRef.current) {
      const height = Math.round(measureElement(containerRef.current).height);
      if (containerHeight !== height) {
        setContainerHeight(height);
      }
    }

    // Measure visible items
    let newHeights: number[] | null = null;
    for (let i = startIndex; i <= endIndex; i++) {
      const itemRef = itemRefs.current[i];
      if (itemRef) {
        const measuredHeight = Math.round(measureElement(itemRef).height);
        if (measuredHeight !== heights[i]) {
          if (!newHeights) {
            newHeights = [...heights];
          }
          newHeights[i] = measuredHeight;
        }
      }
    }

    if (newHeights) {
      setHeights(newHeights);
    }
  });

  // Calculate spacer heights
  const topSpacerHeight = offsets[startIndex] ?? 0;
  const bottomSpacerHeight =
    totalHeight - (offsets[endIndex + 1] ?? totalHeight);

  // Render visible items
  const renderedItems = [];
  for (let i = startIndex; i <= endIndex; i++) {
    const item = data[i];
    if (item) {
      renderedItems.push(
        <Box
          key={keyExtractor(item, i)}
          width="100%"
          ref={(el) => {
            itemRefs.current[i] = el;
          }}
        >
          {renderItem({ item, index: i })}
        </Box>
      );
    }
  }

  return (
    <Box
      ref={containerRef}
      overflowY="scroll"
      overflowX="hidden"
      scrollTop={scrollTop}
      width="100%"
      height="100%"
      flexDirection="column"
    >
      <Box flexShrink={0} width="100%" flexDirection="column">
        <Box height={topSpacerHeight} flexShrink={0} />
        {renderedItems}
        <Box height={bottomSpacerHeight} flexShrink={0} />
      </Box>
    </Box>
  );
}

export default VirtualizedList;
```

---

## Step 10: Helper Utilities

### 10.1 Batched Scroll Hook

Prevents flickering when multiple scroll operations happen in the same tick:

```tsx
import { useRef, useEffect, useCallback } from 'react';

export function useBatchedScroll(currentScrollTop: number) {
  const pendingScrollTopRef = useRef<number | null>(null);
  const currentScrollTopRef = useRef(currentScrollTop);

  useEffect(() => {
    currentScrollTopRef.current = currentScrollTop;
    pendingScrollTopRef.current = null;
  });

  const getScrollTop = useCallback(
    () => pendingScrollTopRef.current ?? currentScrollTopRef.current,
    [],
  );

  const setPendingScrollTop = useCallback((newScrollTop: number) => {
    pendingScrollTopRef.current = newScrollTop;
  }, []);

  return { getScrollTop, setPendingScrollTop };
}
```

**Usage**:
```tsx
const { getScrollTop, setPendingScrollTop } = useBatchedScroll(scrollTop);

// Multiple scrolls in same tick
setPendingScrollTop(10);
setPendingScrollTop(20);
setPendingScrollTop(30);
// Only 30 is applied, preventing 3 separate renders
```

---

## Testing Your Implementation

### Basic Test

```tsx
import { render } from 'ink-testing-library';
import VirtualizedList from './VirtualizedList';

const data = Array.from({ length: 100 }, (_, i) => `Item ${i}`);

const { lastFrame } = render(
  <Box height={10} width={80}>
    <VirtualizedList
      data={data}
      renderItem={({ item }) => <Text>{item}</Text>}
      estimatedItemHeight={() => 1}
      keyExtractor={(item) => item}
    />
  </Box>
);

// Should only show ~10 items, not all 100
const frame = lastFrame();
expect(frame).toContain('Item 0');
expect(frame).not.toContain('Item 50');
```

### Test Scroll Position

```tsx
// Test that scroll position is maintained
// Test that only visible items are rendered
// Test that heights are measured correctly
```

---

## Common Pitfalls & Solutions

### Pitfall 1: Infinite Loops

**Problem**: Component re-renders infinitely

**Solution**: Only update state when values actually change:
```tsx
if (containerHeight !== height) {
  setContainerHeight(height);
}
```

### Pitfall 2: Wrong Visible Range

**Problem**: Items appear/disappear incorrectly

**Solution**: Double-check `startIndex` and `endIndex` calculation:
- `startIndex` should use `-1` buffer
- `endIndex` should check `> scrollTop + containerHeight`

### Pitfall 3: Scroll Position Jumps

**Problem**: Scroll jumps when heights are measured

**Solution**: Use scroll anchor system instead of raw `scrollTop`

### Pitfall 4: Spacer Heights Wrong

**Problem**: Can't scroll to bottom or scroll position is off

**Solution**: Verify spacer calculations:
```tsx
topSpacerHeight = offsets[startIndex]
bottomSpacerHeight = totalHeight - offsets[endIndex + 1]
```

---

## Summary: The Complete Flow

1. **Initialize**: Set up refs, state, and height estimates
2. **Calculate Offsets**: Build cumulative height array
3. **Determine Visible Range**: Find startIndex and endIndex
4. **Render Items**: Only render visible items with refs
5. **Measure Heights**: Use `measureElement` to get actual heights
6. **Update Heights**: Store measured heights in state
7. **Recalculate**: Offsets and visible range update automatically
8. **Render Spacers**: Top and bottom spacers maintain scroll position
9. **Repeat**: Process continues until heights stabilize

**Key Insight**: The component starts with estimates, renders a few items, measures them, updates estimates, and gradually becomes more accurate. This creates a smooth, stable virtualization system!

---

## Next Steps

1. **Add scroll methods**: `scrollBy`, `scrollTo`, `scrollToIndex`
2. **Add "stick to bottom"**: Auto-scroll when new items added
3. **Add keyboard navigation**: Arrow keys, page up/down
4. **Optimize performance**: Memoize callbacks, reduce re-renders
5. **Handle edge cases**: Empty data, very large lists, rapid scrolling

Good luck with your implementation! 🚀

