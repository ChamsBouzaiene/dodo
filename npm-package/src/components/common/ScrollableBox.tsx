/**
 * ScrollableBox - A virtualized scrolling container for Ink
 * 
 * Renders content line by line and shows only visible lines based on scroll position.
 * Supports: Arrow keys, Page Up/Down, g/G navigation.
 */
import React, { useState, useEffect, useCallback, useMemo } from "react";
import { Box, Text, useInput } from "ink";

/**
 * Props for the ScrollableBox component.
 */
export type ScrollableBoxProps = {
  /** Height of the visible area (in terminal rows) */
  height: number;
  /** Whether this component should capture keyboard input for scrolling */
  isActive?: boolean;
  /** Auto-scroll to bottom when content changes */
  autoScroll?: boolean;
  /** Content to render inside the scrollable area */
  children: React.ReactNode;
  /** Show scrollbar indicator */
  showScrollbar?: boolean;
};

// Unicode characters for scrollbar
const SCROLLBAR_TRACK = "│";
const SCROLLBAR_THUMB = "█";

/**
 * Flattens React children into an array of renderable elements,
 * treating each direct child as one "line" for scrolling purposes.
 */
function flattenChildren(children: React.ReactNode): React.ReactNode[] {
  const result: React.ReactNode[] = [];

  React.Children.forEach(children, (child) => {
    if (child === null || child === undefined) return;

    // If it's a fragment, flatten its children
    if (React.isValidElement(child) && child.type === React.Fragment) {
      result.push(...flattenChildren(child.props.children));
    } else {
      result.push(child);
    }
  });

  return result;
}

export const ScrollableBox: React.FC<ScrollableBoxProps> = ({
  height,
  isActive = true,
  autoScroll = true,
  children,
  showScrollbar = true,
}) => {
  const [scrollOffset, setScrollOffset] = useState(0);

  // Flatten children to get individual items
  const items = useMemo(() => flattenChildren(children), [children]);
  const totalItems = items.length;

  // Calculate visible area (reserve 1 line for scroll hint if needed)
  const visibleHeight = Math.max(1, height - 1);

  // Max scroll is total items minus visible height
  const maxScroll = useMemo(() => {
    return Math.max(0, totalItems - visibleHeight);
  }, [totalItems, visibleHeight]);

  // Auto-scroll to bottom when content grows
  useEffect(() => {
    if (autoScroll) {
      setScrollOffset(maxScroll);
    }
  }, [totalItems, autoScroll, maxScroll]);

  // Clamp scroll position when max changes
  useEffect(() => {
    setScrollOffset((prev) => Math.min(prev, maxScroll));
  }, [maxScroll]);

  // Handle keyboard scrolling
  useInput((input, key) => {
    if (!isActive) return;

    const scrollStep = 1;
    const pageStep = Math.max(1, Math.floor(visibleHeight * 0.8));

    if (key.upArrow) {
      setScrollOffset((prev) => Math.max(0, prev - scrollStep));
    } else if (key.downArrow) {
      setScrollOffset((prev) => Math.min(maxScroll, prev + scrollStep));
    } else if (key.pageUp) {
      setScrollOffset((prev) => Math.max(0, prev - pageStep));
    } else if (key.pageDown) {
      setScrollOffset((prev) => Math.min(maxScroll, prev + pageStep));
    } else if (input === "G") {
      setScrollOffset(maxScroll);
    } else if (input === "g") {
      setScrollOffset(0);
    }
  }, { isActive });

  // Get visible items based on scroll position
  const visibleItems = useMemo(() => {
    const startIdx = scrollOffset;
    const endIdx = Math.min(totalItems, startIdx + visibleHeight);
    return items.slice(startIdx, endIdx);
  }, [items, scrollOffset, visibleHeight, totalItems]);

  // Calculate scrollbar
  const scrollbar = useMemo(() => {
    if (!showScrollbar || totalItems <= visibleHeight) {
      return null;
    }

    const trackHeight = visibleHeight;
    const thumbHeight = Math.max(1, Math.round((visibleHeight / totalItems) * trackHeight));
    const scrollRatio = maxScroll > 0 ? scrollOffset / maxScroll : 0;
    const thumbPosition = Math.round(scrollRatio * (trackHeight - thumbHeight));
    const scrollPercent = Math.round(scrollRatio * 100);

    return { trackHeight, thumbHeight, thumbPosition, scrollPercent };
  }, [showScrollbar, totalItems, visibleHeight, scrollOffset, maxScroll]);

  // Render scrollbar column
  const renderScrollbar = useCallback(() => {
    if (!scrollbar) return null;

    const { trackHeight, thumbHeight, thumbPosition } = scrollbar;
    const lines: React.ReactNode[] = [];

    for (let i = 0; i < trackHeight; i++) {
      const isThumb = i >= thumbPosition && i < thumbPosition + thumbHeight;
      lines.push(
        <Text key={i} color={isThumb ? "cyan" : "gray"}>
          {isThumb ? SCROLLBAR_THUMB : SCROLLBAR_TRACK}
        </Text>
      );
    }

    return (
      <Box flexDirection="column" marginLeft={1}>
        {lines}
      </Box>
    );
  }, [scrollbar]);

  // Scroll position indicator
  const scrollHint = useMemo(() => {
    if (!scrollbar) return null;

    const canUp = scrollOffset > 0;
    const canDown = scrollOffset < maxScroll;

    return (
      <Box justifyContent="center">
        <Text color="gray" dimColor>
          {canUp ? "↑" : " "} {scrollbar.scrollPercent}% {canDown ? "↓" : " "} ({totalItems} items)
        </Text>
      </Box>
    );
  }, [scrollbar, scrollOffset, maxScroll, totalItems]);

  // If content fits, just render it directly
  if (totalItems <= visibleHeight) {
    return (
      <Box flexDirection="column" height={height}>
        <Box flexDirection="column" flexGrow={1}>
          {children}
        </Box>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" height={height}>
      <Box flexDirection="row" height={visibleHeight}>
        {/* Visible content */}
        <Box flexDirection="column" flexGrow={1}>
          {visibleItems.map((item, idx) => (
            <Box key={scrollOffset + idx} flexDirection="column">
              {item}
            </Box>
          ))}
        </Box>

        {/* Scrollbar */}
        {renderScrollbar()}
      </Box>

      {/* Scroll hint */}
      {scrollHint}
    </Box>
  );
};

export default ScrollableBox;
