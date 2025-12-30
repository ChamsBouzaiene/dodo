package engine

import (
	"context"
	"errors"
	"testing"
)

// Mock tool function
func mockToolFn(ctx context.Context, args map[string]any) (string, error) {
	if val, ok := args["should_error"]; ok && val.(bool) {
		return "", errors.New("mock error")
	}
	return "success", nil
}

func TestExecuteTool(t *testing.T) {
	ctx := context.Background()
	reg := make(ToolRegistry)

	// Register a mock tool
	reg["mock_tool"] = Tool{
		Name:       "mock_tool",
		Fn:         mockToolFn,
		SchemaJSON: `{"type": "object", "properties": {"should_error": {"type": "boolean"}}}`,
	}

	tests := []struct {
		name    string
		call    ToolCall
		want    string
		wantErr bool
	}{
		{
			name: "success",
			call: ToolCall{
				Name: "mock_tool",
				Args: map[string]any{"should_error": false},
			},
			want:    "success",
			wantErr: false,
		},
		{
			name: "tool execution error",
			call: ToolCall{
				Name: "mock_tool",
				Args: map[string]any{"should_error": true},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "tool not found",
			call: ToolCall{
				Name: "non_existent_tool",
				Args: map[string]any{},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeTool(ctx, tt.call, reg)
			if (err != nil) != tt.wantErr {
				t.Errorf("executeTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("executeTool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteToolCalls(t *testing.T) {
	ctx := context.Background()
	reg := make(ToolRegistry)

	// Register mock tools
	reg["mock_tool"] = Tool{
		Name: "mock_tool",
		Fn:   mockToolFn,
	}
	reg["write_file"] = Tool{ // An edit tool
		Name: "write_file",
		Fn:   mockToolFn,
	}
	reg["write"] = Tool{ // An edit tool
		Name: "write",
		Fn:   mockToolFn,
	}

	tests := []struct {
		name            string
		calls           []ToolCall
		editToolBlocked bool
		wantHistoryLen  int  // Expected number of tool result messages appended
		wantErrorInRes  bool // Check if result contains error message (for blocked tools)
	}{
		{
			name: "single success",
			calls: []ToolCall{
				{ID: "call_1", Name: "mock_tool", Args: map[string]any{}},
			},
			editToolBlocked: false,
			wantHistoryLen:  1,
			wantErrorInRes:  false,
		},
		{
			name: "blocked edit tool (write_file)",
			calls: []ToolCall{
				{ID: "call_2", Name: "write_file", Args: map[string]any{}},
			},
			editToolBlocked: true,
			wantHistoryLen:  1,
			wantErrorInRes:  true, // Should contain error message about planning
		},
		{
			name: "blocked edit tool (write)",
			calls: []ToolCall{
				{ID: "call_2b", Name: "write", Args: map[string]any{}},
			},
			editToolBlocked: true,
			wantHistoryLen:  1,
			wantErrorInRes:  true, // Should contain error message about planning
		},
		{
			name: "mixed allowed and blocked",
			calls: []ToolCall{
				{ID: "call_3", Name: "mock_tool", Args: map[string]any{}},
				{ID: "call_4", Name: "write_file", Args: map[string]any{}},
			},
			editToolBlocked: true,
			wantHistoryLen:  2, // Both get appended (one success, one error)
			wantErrorInRes:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &State{
				Model: "test-model",
			}
			st.EditToolBlocked = tt.editToolBlocked
			hooks := Hooks{} // No-op hooks
			retryConfig := DefaultRetryConfig()

			err := executeToolCalls(ctx, tt.calls, reg, &retryConfig, hooks, st)
			if err != nil {
				t.Fatalf("executeToolCalls() unexpected error: %v", err)
			}

			if len(st.History) != tt.wantHistoryLen {
				t.Errorf("History length = %d, want %d", len(st.History), tt.wantHistoryLen)
			}

			if tt.wantErrorInRes {
				foundError := false
				for _, msg := range st.History {
					if msg.Role == RoleTool && len(msg.Content) > 0 && (msg.Content[0:5] == "ERROR" || msg.Content[0:5] == "valid") {
						// Note: "valid" check is hacky, let's just check for "Planning required"
						// The actual error message starts with "ERROR: Planning required..."
						if len(msg.Content) >= 24 && msg.Content[0:24] == "ERROR: Planning required" {
							foundError = true
							break
						}
					}
				}
				if !foundError {
					t.Error("Expected error message in history for blocked tool, but found none")
				}
			}
		})
	}
}

func TestIsEditTool(t *testing.T) {
	tests := []struct {
		name string
		tool string
		want bool
	}{
		{"write_file", "write_file", true},
		{"write", "write", true},
		{"search_replace", "search_replace", true},
		{"delete_file", "delete_file", true},
		{"read_file", "read_file", false},
		{"grep", "grep", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEditTool(tt.tool); got != tt.want {
				t.Errorf("isEditTool(%q) = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}
