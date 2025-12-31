import React from "react";
import { Box, Text } from "ink";
import { createRequire } from 'module';
const require = createRequire(import.meta.url);
const pkg = require('../../../package.json');

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
░░░░░░░░░░      ░░░░░░░    ░░░░░░░░░░      ░░░░░░░     v${pkg.version}
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


