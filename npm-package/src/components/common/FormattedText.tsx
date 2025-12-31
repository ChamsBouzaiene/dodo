import React from "react";
import { Box, Text } from "ink";
import { lexer } from "marked";

type FormattedTextProps = {
  content: string;
};

/**
 * RenderInline
 * 
 * Recursive component to render inline markdown tokens (text, bold, italic, code, links).
 * Handles the styling of individual text segments.
 */
const RenderInline = ({ tokens }: { tokens?: any[] }) => {
  if (!tokens) return null;

  return (
    <Text>
      {tokens.map((token, i) => {
        if (token.type === "text") {
          // If text has nested tokens (e.g. inside a link or just mixed), recurse
          if (token.tokens) {
            return <RenderInline key={i} tokens={token.tokens} />;
          }
          // Decode simple HTML entities if needed, but for now just raw text
          return <Text key={i}>{token.text}</Text>;
        }
        if (token.type === "strong") {
          return (
            <Text key={i} bold>
              <RenderInline tokens={token.tokens} />
            </Text>
          );
        }
        if (token.type === "em") {
          return (
            <Text key={i} italic>
              <RenderInline tokens={token.tokens} />
            </Text>
          );
        }
        if (token.type === "codespan") {
          return (
            <Text key={i} color="blue" backgroundColor="black">
              {token.text}
            </Text>
          );
        }
        if (token.type === "link") {
          return <Text key={i} color="blue" underline>{token.text}</Text>;
        }
        // Fallback
        return <Text key={i}>{token.raw}</Text>;
      })}
    </Text>
  );
};

/**
 * FormattedText
 * 
 * Renders markdown content using Ink components.
 * Supports paragraphs, code blocks, lists, headings, and inline formatting.
 * Uses `marked.lexer` to parse the markdown string into tokens.
 */
export const FormattedText: React.FC<FormattedTextProps> = ({ content }) => {
  if (!content) return null;

  const tokens = lexer(content);

  return (
    <Box flexDirection="column">
      {tokens.map((token: any, index: number) => {
        if (token.type === "paragraph") {
          return (
            <Box key={index} marginBottom={1}>
              <RenderInline tokens={token.tokens} />
            </Box>
          );
        }

        if (token.type === "code") {
          return (
            <Box
              key={index}
              flexDirection="column"
              marginBottom={1}
              borderStyle="round"
              borderColor="gray"
              paddingX={1}
            >
              <Text color="yellow">{token.text}</Text>
            </Box>
          );
        }

        if (token.type === "list") {
          return (
            <Box key={index} flexDirection="column" marginBottom={1}>
              {token.items.map((item: any, i: number) => (
                <Box key={i} marginLeft={1}>
                  <Text color="green">• </Text>
                  <RenderInline tokens={item.tokens} />
                </Box>
              ))}
            </Box>
          );
        }

        if (token.type === "heading") {
          return (
            <Box key={index} marginBottom={1} marginTop={1}>
              <Text bold underline>{token.text}</Text>
            </Box>
          )
        }

        if (token.type === "space") {
          return null;
        }

        // Fallback for unknown blocks
        return (
          <Box key={index} marginBottom={1}>
            {token.text && <Text>{token.text}</Text>}
          </Box>
        );
      })}
    </Box>
  );
};
