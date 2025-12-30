package engine

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

type ToolFunc func(ctx context.Context, args map[string]any) (string, error)

// ToolMetadata provides versioning and categorization for tools.
type ToolMetadata struct {
	Version         string   // e.g., "1.0.0"
	Category        string   // e.g., "filesystem", "network", "analysis"
	Tags            []string // e.g., ["read-only", "idempotent"]
	Author          string   // Optional
	Deprecated      bool     // Mark for removal
	ReplacedBy      string   // Tool name that replaces this one
	MinAgentVersion string   // Minimum engine version required
}

type Tool struct {
	Name        string
	Description string
	SchemaJSON  string
	Fn          ToolFunc
	Retryable   bool // Whether this tool can be retried (default: true for idempotent tools)
	Metadata    ToolMetadata
}

// ValidateArgs validates the provided arguments against the tool's JSON schema.
func (t Tool) ValidateArgs(args map[string]any) error {
	// Use gojsonschema for validation
	schemaLoader := gojsonschema.NewStringLoader(t.SchemaJSON)
	documentLoader := gojsonschema.NewGoLoader(args)
	
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	
	if !result.Valid() {
		var errorMsgs []string
		for _, err := range result.Errors() {
			errorMsgs = append(errorMsgs, err.String())
		}
		return &ToolValidationError{
			ToolName: t.Name,
			Errors:   errorMsgs,
		}
	}
	
	return nil
}

type ToolRegistry map[string]Tool

// IsDeprecated returns true if this tool is marked as deprecated.
func (t Tool) IsDeprecated() bool {
	return t.Metadata.Deprecated
}

// GetVersion returns the tool version, defaulting to "0.0.0" if unset.
func (t Tool) GetVersion() string {
	if t.Metadata.Version == "" {
		return "0.0.0"
	}
	return t.Metadata.Version
}

// GetCategory returns the tool category, defaulting to "general" if unset.
func (t Tool) GetCategory() string {
	if t.Metadata.Category == "" {
		return "general"
	}
	return t.Metadata.Category
}

func (r ToolRegistry) Schemas() []ToolSchema {
	s := make([]ToolSchema, 0, len(r))
	for _, t := range r {
		// Note: Since Retryable is a bool, zero value is false.
		// We can't distinguish "unset" from "explicitly false".
		// Tools should explicitly set Retryable=false for non-idempotent operations.
		// For now, we use the value as-is (defaults to false, but retry logic will handle it).
		s = append(s, ToolSchema{
			Name:        t.Name,
			Description: t.Description,
			JSONSchema:  t.SchemaJSON,
			Retryable:   t.Retryable,
		})
	}
	return s
}

// FilterByCategory returns a new registry containing only tools of the given category.
func (r ToolRegistry) FilterByCategory(category string) ToolRegistry {
	filtered := make(ToolRegistry)
	for name, tool := range r {
		if tool.GetCategory() == category {
			filtered[name] = tool
		}
	}
	return filtered
}

// ListDeprecated returns a list of deprecated tool names.
func (r ToolRegistry) ListDeprecated() []string {
	var deprecated []string
	for name, tool := range r {
		if tool.IsDeprecated() {
			deprecated = append(deprecated, name)
		}
	}
	return deprecated
}
