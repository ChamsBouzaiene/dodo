export type ToolConfig = {
  icon: string;
  label: (metadata: Record<string, any>) => string;
  color: string;
  category: string;
};

export const TOOL_CONFIGS: Record<string, ToolConfig> = {
  // Execution tools
  run_cmd: {
    icon: "",
    label: (m) => {
      const cmd = m.command || m.cmd || "";
      if (cmd.length > 50) {
        return `run: ${cmd.substring(0, 47)}...`;
      }
      return cmd ? `run: ${cmd}` : "Run command";
    },
    color: "cyan",
    category: "execution",
  },
  run_tests: {
    icon: "",
    label: () => "Run tests",
    color: "green",
    category: "execution",
  },
  run_build: {
    icon: "",
    label: () => "Build",
    color: "yellow",
    category: "execution",
  },
  run_terminal_cmd: {
    icon: "",
    label: (m) => {
      const cmd = m.command || "";
      if (cmd.length > 50) {
        return `run: ${cmd.substring(0, 47)}...`;
      }
      return cmd ? `run: ${cmd}` : "Run terminal command";
    },
    color: "cyan",
    category: "execution",
  },

  // Editing tools
  search_replace: {
    icon: "",
    label: (m) => {
      const file = m.file || m.file_path || "";
      return file ? `Edit ${file}` : "Edit file";
    },
    color: "blue",
    category: "editing",
  },
  write: {
    icon: "",
    label: (m) => {
      const file = m.file || m.file_path || "";
      return file ? `Write ${file}` : "Write file";
    },
    color: "blue",
    category: "editing",
  },
  write_file: {
    icon: "",
    label: (m) => {
      const file = m.file || m.path || "";
      return file ? `Write ${file}` : "Write file";
    },
    color: "blue",
    category: "editing",
  },
  propose_diff: {
    icon: "",
    label: (m) => {
      const file = m.file || m.file_path || "";
      return file ? `Propose diff for ${file}` : "Propose diff";
    },
    color: "blue",
    category: "editing",
  },

  // Search tools
  codebase_search: {
    icon: "",
    label: (m) => {
      const query = m.query || "";
      if (query.length > 40) {
        return `Search: ${query.substring(0, 37)}...`;
      }
      return query ? `Search: ${query}` : "Codebase search";
    },
    color: "magenta",
    category: "search",
  },
  grep: {
    icon: "",
    label: (m) => {
      const pattern = m.pattern || "";
      if (pattern.length > 40) {
        return `Grep: ${pattern.substring(0, 37)}...`;
      }
      return pattern ? `Grep: ${pattern}` : "Grep";
    },
    color: "magenta",
    category: "search",
  },

  // Filesystem tools
  read_file: {
    icon: "",
    label: (m) => {
      const path = m.path || "";
      return path ? `Read ${path}` : "Read file";
    },
    color: "gray",
    category: "filesystem",
  },
  read_span: {
    icon: "",
    label: (m) => {
      const path = m.path || "";
      const start = m.start_line || m.start;
      const end = m.end_line || m.end;
      if (path && start && end) {
        return `Read ${path}:${start}-${end}`;
      }
      return path ? `Read ${path}` : "Read span";
    },
    color: "gray",
    category: "filesystem",
  },
  delete_file: {
    icon: "",
    label: (m) => {
      const path = m.path || "";
      return path ? `Delete ${path}` : "Delete file";
    },
    color: "red",
    category: "filesystem",
  },
  list_files: {
    icon: "",
    label: (m) => {
      const path = m.path || "";
      return path ? `List ${path}` : "List files";
    },
    color: "gray",
    category: "filesystem",
  },

  // Meta tools
  think: {
    icon: "",
    label: (m) => {
      const summary = m.summary || "";
      return summary || "Planning next action";
    },
    color: "dim",
    category: "meta",
  },
  respond: {
    icon: "",
    label: () => "Final answer",
    color: "green",
    category: "meta",
  },
  plan: {
    icon: "",
    label: () => "Create execution plan",
    color: "blue",
    category: "meta",
  },
  revise_plan: {
    icon: "",
    label: () => "Revise plan",
    color: "blue",
    category: "meta",
  },
};

export function getToolConfig(toolName: string): ToolConfig {
  // Handle reasoning steps (empty tool name)
  if (!toolName || toolName === "") {
    return {
      icon: "",
      label: () => "Reasoning",
      color: "blue",
      category: "reasoning",
    };
  }

  return (
    TOOL_CONFIGS[toolName] || {
      icon: "",
      label: () => toolName,
      color: "gray",
      category: "unknown",
    }
  );
}

export function formatToolLabel(toolName: string, metadata: Record<string, any>): string {
  const config = getToolConfig(toolName);
  return config.label(metadata || {});
}

