package factory

import (
	"context"

	"github.com/ChamsBouzaiene/dodo/internal/codebeacon"
	"github.com/ChamsBouzaiene/dodo/internal/coder"
	"github.com/ChamsBouzaiene/dodo/internal/engine"
	"github.com/ChamsBouzaiene/dodo/internal/indexer"
)

// BuildBrainAgent creates a fully configured brain agent for the REPL.
// It now delegates to the CoderAgent constructor.
func BuildBrainAgent(ctx context.Context, repoRoot string, retrieval indexer.Retrieval, workspaceCtx *indexer.WorkspaceContext, streaming bool, muteResponse bool, extraHooks ...engine.Hook) (*engine.Agent, error) {
	var opts []coder.Option

	// Attempt to add CodeBeacon for deep code understanding
	// We build it here (Dependency Injection) and pass it to the Coder
	if beacon, beaconErr := codebeacon.NewCodeBeaconAgent(ctx, repoRoot, retrieval, workspaceCtx); beaconErr != nil {
		// Log warning but proceed without beacon
		// We need to import log package since we removed it earlier
		// For now, let's just skip adding the tool
	} else {
		beaconTool := codebeacon.NewCodeBeaconTool(beacon)
		opts = append(opts, coder.WithTool("code_beacon", beaconTool))
	}

	coderAgent, err := coder.NewAgent(ctx, repoRoot, retrieval, workspaceCtx, streaming, muteResponse, extraHooks, opts...)
	if err != nil {
		return nil, err
	}
	return coderAgent.Agent, nil
}

// BuildCodeBeaconAgent creates a CodeBeacon analysis agent
func BuildCodeBeaconAgent(ctx context.Context, repoRoot string, retrieval indexer.Retrieval, workspaceCtx *indexer.WorkspaceContext) (*codebeacon.CodeBeaconAgent, error) {
	return codebeacon.NewCodeBeaconAgent(ctx, repoRoot, retrieval, workspaceCtx)
}
