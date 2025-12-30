package patch

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// PatchResult represents a patch to be applied to a file.
// It contains either a unified diff or new content (or both).
type PatchResult struct {
	Path        string `json:"path"`
	NewContent  string `json:"new_content"`
	UnifiedDiff string `json:"unified_diff"`
}

// Apply applies a PatchResult to repoRoot and returns a human-readable status string.
// It only knows:
//   - repoRoot (string)
//   - PatchResult (path/newContent/diff)
//   - filesystem + `patch` binary
//
// No schema.Message, no Eino, no LLM.
func Apply(ctx context.Context, repoRoot string, p PatchResult) (string, error) {
	// Prefer unified diff if available, otherwise fall back to new content
	diff := p.UnifiedDiff
	if diff == "" && p.NewContent != "" {
		// If we only have new content, we'll use direct write fallback
		diff = ""
	}

	if diff == "" && p.NewContent == "" {
		return "", fmt.Errorf("patch result has neither unified_diff nor new_content")
	}

	if p.Path == "" {
		return "", fmt.Errorf("patch result missing path")
	}

	if len(diff) > 500 {
		log.Printf("Applying diff (first 500 chars):\n%s...", diff[:500])
	} else if diff != "" {
		log.Printf("Applying diff:\n%s", diff)
	}

	// If we have a unified diff, try to apply it with patch binary
	if diff != "" {
		tmpFile, err := os.CreateTemp("", "dodo-diff-*.patch")
		if err != nil {
			return "", fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(diff); err != nil {
			tmpFile.Close()
			return "", fmt.Errorf("failed to write diff to temp file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			return "", fmt.Errorf("failed to close temp file: %w", err)
		}

		// Try to apply patch with dry-run first (be tolerant to small context/whitespace diffs)
		cmd := exec.CommandContext(ctx, "patch", "-p0", "--dry-run", "--fuzz=2", "--ignore-whitespace", "-i", tmpFile.Name())
		cmd.Dir = repoRoot
		dryRunOutput, dryRunErr := cmd.CombinedOutput()

		if dryRunErr != nil {
			// Patch failed - try direct write fallback if we have the content
			if p.NewContent != "" && p.Path != "" {
				log.Printf("⚠️  patch dry-run failed for %s, falling back to direct write: %v", p.Path, dryRunErr)
				log.Printf("Dry-run output:\n%s", string(dryRunOutput))

				return applyDirectWrite(repoRoot, p.Path, p.NewContent)
			}

			if p.NewContent == "" {
				log.Printf("❌ Cannot use fallback: new_content field missing from patch result")
				log.Printf("Patch result must include 'new_content' field for fallback")
			}
			errorMsg := fmt.Sprintf("patch validation failed: %v\n\nDry-run output:\n%s\n\nDiff content:\n%s\n\nNewContent length: %d bytes", dryRunErr, string(dryRunOutput), diff, len(p.NewContent))
			log.Printf("Patch error details:\n%s", errorMsg)
			return errorMsg, fmt.Errorf("patch dry-run failed: %w", dryRunErr)
		}

		// Dry run passed, apply patch (same tolerant flags)
		cmd = exec.CommandContext(ctx, "patch", "-p0", "--fuzz=2", "--ignore-whitespace", "-i", tmpFile.Name())
		cmd.Dir = repoRoot
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("patch application failed: %v\nOutput: %s\nDiff content:\n%s", err, string(output), diff), fmt.Errorf("patch failed: %w", err)
		}

		return fmt.Sprintf("diff applied successfully\n%s", string(output)), nil
	}

	// No diff available, use direct write
	if p.NewContent == "" {
		return "", fmt.Errorf("no diff or new content available")
	}

	return applyDirectWrite(repoRoot, p.Path, p.NewContent)
}

// applyDirectWrite writes new content directly to a file.
func applyDirectWrite(repoRoot, filePath, newContent string) (string, error) {
	fullPath := fmt.Sprintf("%s/%s", repoRoot, filePath)
	if err := os.WriteFile(fullPath, []byte(newContent), 0o644); err != nil {
		return "", fmt.Errorf("fallback write failed: %w", err)
	}

	log.Printf("✅ Successfully wrote new content directly to %s", filePath)
	return fmt.Sprintf("wrote new content directly to %s (patch failed, used fallback)", filePath), nil
}
