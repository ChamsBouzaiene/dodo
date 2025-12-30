package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/ChamsBouzaiene/dodo/internal/engine"
	"github.com/ChamsBouzaiene/dodo/internal/factory"
	"github.com/ChamsBouzaiene/dodo/internal/indexer"
)

func main() {
	// Load .env file if it exists (same as main dodo command)
	_ = godotenv.Load()

	ctx := context.Background()

	args := os.Args[1:]
	if len(args) > 0 && args[0] == "engine" {
		if err := runEngineCommand(ctx, args[1:]); err != nil {
			// Check if we're in stdio mode by looking for --stdio flag
			stdioMode := false
			for _, arg := range args[1:] {
				if arg == "--stdio" {
					stdioMode = true
					break
				}
			}

			if stdioMode {
				// In stdio mode, print to stderr to avoid corrupting the protocol
				fmt.Fprintf(os.Stderr, "FATAL: engine command failed: %v\n", err)
				os.Exit(1)
			} else {
				log.Fatalf("engine command failed: %v", err)
			}
		}
		return
	}

	if err := runDefaultCommand(ctx, args); err != nil {
		log.Fatalf("command failed: %v", err)
	}
}

func runDefaultCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("dodo", flag.ExitOnError)
	enableStreaming := fs.Bool("stream", false, "Enable streaming mode for incremental output")
	repoFlag := fs.String("repo", "", "Path to repository root (default: current directory)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	env, err := prepareRuntimeEnv(ctx, *repoFlag)
	if err != nil {
		return err
	}
	defer env.Close()

	runBrainMode(ctx, env.RepoRoot, env.Retrieval, env.WorkspaceCtx, *enableStreaming)
	return nil
}

func runEngineCommand(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("engine", flag.ExitOnError)
	repoFlag := fs.String("repo", "", "Path to repository root (default: current directory)")
	enableStreaming := fs.Bool("stream", true, "Enable streaming mode when serving over stdio")
	stdioMode := fs.Bool("stdio", false, "Serve the engine over the NDJSON stdio protocol")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Redirect logs to stderr in stdio mode to avoid corrupting the protocol
	if *stdioMode {
		log.SetOutput(os.Stderr)
	}

	env, err := prepareRuntimeEnv(ctx, *repoFlag)
	if err != nil {
		// In stdio mode, print errors to stderr so they don't corrupt the protocol
		if *stdioMode {
			fmt.Fprintf(os.Stderr, "ERROR: failed to prepare runtime environment: %v\n", err)
		}
		return err
	}
	defer env.Close()

	if *stdioMode {
		return runStdIOEngine(ctx, env, *enableStreaming)
	}

	runBrainMode(ctx, env.RepoRoot, env.Retrieval, env.WorkspaceCtx, *enableStreaming)
	return nil
}

func runBrainMode(ctx context.Context, absRepoRoot string, retrieval indexer.Retrieval, workspaceCtx *indexer.WorkspaceContext, streaming bool) {
	log.Println("🧠 Starting brain agent (interactive mode)")

	brainAgent, err := factory.BuildBrainAgent(ctx, absRepoRoot, retrieval, workspaceCtx, streaming, false)
	if err != nil {
		log.Fatalf("failed to create brain agent: %v", err)
	}

	log.Printf("Brain agent ready (streaming: %v, repo: %s)", streaming, absRepoRoot)
	log.Printf("🧩 Internal planning enforcement is active (plan tool required before edits)")
	log.Printf("🔦 Use 'code_beacon' for deep codebase investigations when needed")

	s := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("you> ")
		if !s.Scan() {
			break
		}
		line := s.Text()
		if line == "" {
			continue
		}

		if err := brainAgent.Run(ctx, line); err != nil {
			if engine.IsSoftCapError(err) {
				log.Printf("⚠️  %v", err)
			} else {
				log.Printf("error: %v", err)
			}
		}
		fmt.Println()
	}
}
