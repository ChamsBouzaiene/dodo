package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ChamsBouzaiene/dodo/internal/sandbox"
)

// MockRunner implements execution.Runner for testing
type MockRunner struct {
	RunCmdFunc func(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (sandbox.Result, error)
}

func (m *MockRunner) RunCmd(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (sandbox.Result, error) {
	if m.RunCmdFunc != nil {
		return m.RunCmdFunc(ctx, repoDir, name, args, timeout)
	}
	return sandbox.Result{}, nil
}

func TestGrepImpl(t *testing.T) {
	tests := []struct {
		name            string
		pattern         string
		path            string
		globs           string
		caseInsensitive bool
		mockStdout      string
		mockStderr      string
		mockExitCode    int
		mockErr         error
		wantResults     int
		wantTruncated   bool
		wantErr         bool
	}{
		{
			name:    "Basic match",
			pattern: "func main",
			mockStdout: `{"type":"match","data":{"path":{"text":"main.go"},"lines":{"text":"func main() {"},"line_number":10,"submatches":[{"match":{"text":"func main"},"start":0,"end":9}]}}
{"type":"match","data":{"path":{"text":"cmd/app.go"},"lines":{"text":"func main() {"},"line_number":5,"submatches":[{"match":{"text":"func main"},"start":0,"end":9}]}}`,
			wantResults: 2,
		},
		{
			name:         "No matches",
			pattern:      "foobar",
			mockExitCode: 1, // rg returns 1 for no matches
			mockErr:      fmt.Errorf("exit status 1"),
			wantResults:  0,
		},
		{
			name:    "Ignore non-match lines",
			pattern: "foo",
			mockStdout: `{"type":"begin","data":{"path":{"text":"main.go"}}}
{"type":"match","data":{"path":{"text":"main.go"},"lines":{"text":"foo"},"line_number":1,"submatches":[]}}
{"type":"end","data":{"path":{"text":"main.go"},"binary_offset":null,"stats":{"elapsed":{"secs":0,"nanos":1000},"searches":1,"searches_with_match":1,"bytes_searched":100,"bytes_printed":100,"matched_lines":1,"matches":1}}}`,
			wantResults: 1,
		},
		{
			name:          "Truncated results",
			pattern:       "common",
			mockStdout:    generateMockOutput(150), // Generate 150 matches
			wantResults:   100,
			wantTruncated: true,
		},
		{
			name:    "rg error",
			pattern: "invalid(",
			mockErr: fmt.Errorf("command failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &MockRunner{
				RunCmdFunc: func(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (sandbox.Result, error) {
					// Verify args
					if name != "rg" {
						t.Errorf("expected command rg, got %s", name)
					}

					// Basic arg validation
					hasJson := false
					hasPattern := false
					for i, arg := range args {
						if arg == "--json" {
							hasJson = true
						}
						if arg == "-e" && i+1 < len(args) && args[i+1] == tt.pattern {
							hasPattern = true
						}
					}
					if !hasJson {
						t.Error("expected --json flag")
					}
					if !hasPattern {
						t.Errorf("expected pattern %s in args", tt.pattern)
					}

					return sandbox.Result{
						Stdout: tt.mockStdout,
						Stderr: tt.mockStderr,
						Code:   tt.mockExitCode,
					}, tt.mockErr
				},
			}

			resultJSON, err := grepImpl(context.Background(), runner, "/repo", tt.pattern, tt.path, tt.globs, tt.caseInsensitive)
			if (err != nil) != tt.wantErr {
				t.Errorf("grepImpl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
					t.Fatalf("failed to unmarshal result: %v", err)
				}

				results := result["results"].([]interface{})
				if len(results) != tt.wantResults {
					t.Errorf("got %d results, want %d", len(results), tt.wantResults)
				}

				if tt.wantTruncated {
					if truncated, ok := result["truncated"].(bool); !ok || !truncated {
						t.Error("expected truncated=true")
					}
				}
			}
		})
	}
}

func generateMockOutput(count int) string {
	var sb strings.Builder
	for i := 0; i < count; i++ {
		sb.WriteString(fmt.Sprintf(`{"type":"match","data":{"path":{"text":"file%d.go"},"lines":{"text":"match %d"},"line_number":%d,"submatches":[]}}`+"\n", i, i, i))
	}
	return sb.String()
}
