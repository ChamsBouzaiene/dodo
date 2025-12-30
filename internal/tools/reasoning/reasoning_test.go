package reasoning

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

func TestThinkTool(t *testing.T) {
	tool := NewThinkTool()

	tests := []struct {
		name    string
		args    map[string]any
		wantErr bool
	}{
		{
			name: "Valid reasoning",
			args: map[string]any{
				"reasoning": "I am thinking about code",
			},
			wantErr: false,
		},
		{
			name: "Valid reason alias",
			args: map[string]any{
				"reason": "I am thinking about code",
			},
			wantErr: false,
		},
		{
			name: "Empty reasoning",
			args: map[string]any{
				"reasoning": "",
			},
			wantErr: true,
		},
		{
			name:    "Missing arguments",
			args:    map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Fn(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ThinkTool.Fn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				var res map[string]interface{}
				if err := json.Unmarshal([]byte(result), &res); err != nil {
					t.Errorf("Failed to unmarshal result: %v", err)
				}
				if res["status"] != "noted" {
					t.Errorf("Expected status 'noted', got %v", res["status"])
				}
			}
		})
	}
}

func TestRespondTool(t *testing.T) {
	tool := NewRespondTool()

	tests := []struct {
		name    string
		args    map[string]any
		wantErr bool
	}{
		{
			name: "Valid response",
			args: map[string]any{
				"summary": "Task complete",
			},
			wantErr: false,
		},
		{
			name: "Valid response with details",
			args: map[string]any{
				"summary":       "Task complete",
				"files_changed": []interface{}{"file1.go", "file2.go"},
				"next_steps":    []interface{}{"Run tests"},
			},
			wantErr: false,
		},
		{
			name: "Missing summary",
			args: map[string]any{
				"files_changed": []interface{}{"file1.go"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Fn(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("RespondTool.Fn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				var res RespondResult
				if err := json.Unmarshal([]byte(result), &res); err != nil {
					t.Errorf("Failed to unmarshal result: %v", err)
				}
				if res.Status != "complete" {
					t.Errorf("Expected status 'complete', got %v", res.Status)
				}
				if res.Summary == "" {
					t.Error("Expected summary in result")
				}
			}
		})
	}
}

func TestPlanTool(t *testing.T) {
	tool := NewPlanTool()

	tests := []struct {
		name    string
		args    map[string]any
		wantErr bool
	}{
		{
			name: "Valid plan",
			args: map[string]any{
				"task_summary": "Refactor auth",
				"steps": []interface{}{
					map[string]interface{}{"id": "1", "description": "Step 1"},
					map[string]interface{}{"id": "2", "description": "Step 2"},
					map[string]interface{}{"id": "3", "description": "Step 3"},
				},
				"target_areas": []interface{}{"auth"},
			},
			wantErr: false,
		},
		{
			name: "Too few steps",
			args: map[string]any{
				"task_summary": "Refactor auth",
				"steps": []interface{}{
					map[string]interface{}{"id": "1", "description": "Step 1"},
					map[string]interface{}{"id": "2", "description": "Step 2"},
				},
				"target_areas": []interface{}{"auth"},
			},
			wantErr: true,
		},
		{
			name: "Missing required fields",
			args: map[string]any{
				"task_summary": "Refactor auth",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock engine state
			st := &engine.State{
				EditToolBlocked: true,
			}
			ctx := context.WithValue(context.Background(), "engine_state", st)

			result, err := tool.Fn(ctx, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("PlanTool.Fn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if st.MiniPlan == nil {
					t.Error("Expected MiniPlan to be set in state")
				}
				if st.EditToolBlocked {
					t.Error("Expected EditToolBlocked to be false")
				}
				if !strings.Contains(result, "Internal execution plan created") {
					t.Error("Expected success message")
				}
			}
		})
	}
}

func TestRevisePlanTool(t *testing.T) {
	tool := NewRevisePlanTool()

	tests := []struct {
		name      string
		setupPlan *engine.MiniPlan
		args      map[string]any
		wantErr   bool
	}{
		{
			name: "Valid revision",
			setupPlan: &engine.MiniPlan{
				Steps: []engine.MiniPlanStep{
					{ID: "1", Description: "Old Step 1", Status: engine.StepStatusCompleted},
				},
			},
			args: map[string]any{
				"reason": "Need change",
				"revised_steps": []interface{}{
					map[string]interface{}{"id": "1", "description": "New Step 1"},
					map[string]interface{}{"id": "2", "description": "New Step 2"},
				},
			},
			wantErr: false,
		},
		{
			name:      "No existing plan",
			setupPlan: nil,
			args: map[string]any{
				"reason": "Need change",
				"revised_steps": []interface{}{
					map[string]interface{}{"id": "1", "description": "New Step 1"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &engine.State{
				MiniPlan: tt.setupPlan,
			}
			ctx := context.WithValue(context.Background(), "engine_state", st)

			_, err := tool.Fn(ctx, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("RevisePlanTool.Fn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(st.MiniPlan.Revisions) != 1 {
					t.Error("Expected 1 revision record")
				}
				if len(st.MiniPlan.Steps) != 2 {
					t.Errorf("Expected 2 steps, got %d", len(st.MiniPlan.Steps))
				}
				// Check status preservation
				if st.MiniPlan.Steps[0].Status != engine.StepStatusCompleted {
					t.Error("Expected step 1 status to be preserved as Completed")
				}
			}
		})
	}
}

func TestProjectPlanTool(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewProjectPlanTool(tmpDir)

	// Test Update (Create)
	_, err := tool.Fn(context.Background(), map[string]any{
		"mode":    "update",
		"content": "Initial plan",
	})
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	// Test Read
	content, err := tool.Fn(context.Background(), map[string]any{
		"mode": "read",
	})
	if err != nil {
		t.Fatalf("Failed to read plan: %v", err)
	}
	if !strings.Contains(content, "Initial plan") {
		t.Error("Read content mismatch")
	}

	// Test Append
	_, err = tool.Fn(context.Background(), map[string]any{
		"mode":    "append",
		"content": "New item",
	})
	if err != nil {
		t.Fatalf("Failed to append plan: %v", err)
	}

	// Verify Append
	content, _ = tool.Fn(context.Background(), map[string]any{"mode": "read"})
	if !strings.Contains(content, "New item") {
		t.Error("Appended content missing")
	}
}
