import React from "react";
import { Box, Text } from "ink";
// @ts-ignore
const version = process.env.CLI_VERSION || '0.0.0';

export const Header: React.FC = () => {
  return (
    <Box flexDirection="column" paddingX={1} marginBottom={1}>
      <Box flexDirection="row" alignItems="center">
        <Box paddingY={1} marginRight={4}>
          <Text color="cyan" bold>
            {`
 ██████████      ███████    ██████████      ███████   
░░███░░░░███   ███░░░░░███ ░░███░░░░███   ███░░░░░███ 
 ░███   ░░███ ███     ░░███ ░███   ░░███ ███     ░░███
 ░███    ░███░███      ░███ ░███    ░███░███      ░███
 ░███    ░███░███      ░███ ░███    ░███░███      ░███
 ░███    ███ ░░███     ███  ░███    ███ ░░███     ███ 
 ██████████   ░░░███████░   ██████████   ░░░███████░     
░░░░░░░░░░      ░░░░░░░    ░░░░░░░░░░      ░░░░░░░     v${version}
 `}


          </Text>
        </Box>
      </Box>
      <Text color="yellow" italic>
        The Agentic AI Coding Assistant
      </Text>
    </Box >
  );
};


