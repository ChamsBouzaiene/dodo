package engine

// ToolSet specifies which categories of tools to include in the registry.
type ToolSet struct {
	Filesystem bool // read_file, list_files, write_file, delete_file
	Search     bool // grep, codebase_search, read_span
	Execution  bool // run_tests, run_build, run_cmd
	Editing    bool // search_replace, write
	Semantic   bool // Requires retrieval - codebase_search, read_span
	Meta       bool // think (reasoning and thought process), respond (task completion)
}
