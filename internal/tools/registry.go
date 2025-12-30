package tools

import (
	"github.com/ChamsBouzaiene/dodo/internal/engine"
	"github.com/ChamsBouzaiene/dodo/internal/indexer"
	"github.com/ChamsBouzaiene/dodo/internal/tools/editing"
	"github.com/ChamsBouzaiene/dodo/internal/tools/execution"
	"github.com/ChamsBouzaiene/dodo/internal/tools/filesystem"
	"github.com/ChamsBouzaiene/dodo/internal/tools/reasoning"
	"github.com/ChamsBouzaiene/dodo/internal/tools/search"
)

// NewToolRegistry creates a new engine.ToolRegistry based on the provided ToolSet.
// It copies implementations from internal/tools/*.go and wraps them as engine.Tool.
func NewToolRegistry(repoRoot string, retrieval indexer.Retrieval, set engine.ToolSet) (engine.ToolRegistry, error) {
	reg := make(engine.ToolRegistry)

	if set.Filesystem {
		reg["read_file"] = filesystem.NewReadFileTool(repoRoot)
		reg["list_files"] = filesystem.NewListFilesTool(repoRoot)
		reg["write_file"] = filesystem.NewWriteFileTool(repoRoot)
		reg["delete_file"] = filesystem.NewDeleteFileTool(repoRoot)
	}

	if set.Search {
		reg["grep"] = search.NewGrepTool(repoRoot)
		if retrieval != nil && set.Semantic {
			reg["codebase_search"] = search.NewCodebaseSearchTool(retrieval)
			reg["read_span"] = search.NewReadSpanTool(retrieval)
		}
	}

	if set.Execution {
		reg["run_tests"] = execution.NewRunTestsTool(repoRoot)
		reg["run_build"] = execution.NewRunBuildTool(repoRoot)
		reg["run_cmd"] = execution.NewRunCmdTool(repoRoot)
	}

	if set.Editing {
		reg["search_replace"] = editing.NewSearchReplaceTool(repoRoot)
		reg["write"] = editing.NewWriteTool(repoRoot)
	}

	if set.Meta {
		reg["think"] = reasoning.NewThinkTool()
		reg["respond"] = reasoning.NewRespondTool()
	}

	return reg, nil
}
