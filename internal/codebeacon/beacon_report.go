package codebeacon

import (
	"fmt"
	"strings"
)

// BeaconReport is the structured output from CodeBeacon analysis.
// It provides a comprehensive understanding of codebase structure and patterns
// to help the brain agent make informed implementation decisions.
type BeaconReport struct {
	InvestigationGoal string         `json:"investigation_goal"`
	Summary           string         `json:"summary"`         // 2-3 paragraph overview
	RelevantFiles     []FileAnalysis `json:"relevant_files"`  // Key files found
	KeyTypes          []TypeInfo     `json:"key_types"`       // Important interfaces/types
	Dependencies      []Dependency   `json:"dependencies"`    // Call graph fragments
	Patterns          []Pattern      `json:"patterns"`        // Observed code patterns
	Risks             []string       `json:"risks"`           // Identified risks
	Recommendations   []string       `json:"recommendations"` // Suggested approach
	FilesRead         []string       `json:"files_read"`      // Files already analyzed (for brain agent to avoid re-reading)
}

// FileAnalysis describes a file relevant to the investigation.
type FileAnalysis struct {
	Path       string   `json:"path"`
	Relevance  string   `json:"relevance"`   // Why this file matters
	KeySymbols []string `json:"key_symbols"` // Important functions/types
}

// TypeInfo describes an important type, interface, or function.
type TypeInfo struct {
	Name            string   `json:"name"`
	Kind            string   `json:"kind"` // "interface", "struct", "function"
	Location        string   `json:"location"`
	Implementations []string `json:"implementations"` // For interfaces
}

// Dependency represents a relationship between code elements.
type Dependency struct {
	From string `json:"from"` // "PackageA.FunctionX"
	To   string `json:"to"`   // "PackageB.FunctionY"
	Type string `json:"type"` // "calls", "implements", "uses"
}

// Pattern describes an architectural or code pattern observed in the codebase.
type Pattern struct {
	Name        string   `json:"name"`        // "Middleware Pattern"
	Description string   `json:"description"` // How it's implemented
	Examples    []string `json:"examples"`    // File paths showing pattern
}

// FormatForAgent returns a human-readable markdown report for the brain agent.
// This is what gets returned as the tool result when code_beacon is called.
func (r *BeaconReport) FormatForAgent() string {
	var sb strings.Builder

	sb.WriteString("# CodeBeacon Analysis Report\n\n")
	sb.WriteString(fmt.Sprintf("**Investigation Goal**: %s\n\n", r.InvestigationGoal))

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString(r.Summary)
	sb.WriteString("\n\n")

	// Relevant Files
	if len(r.RelevantFiles) > 0 {
		sb.WriteString("## Relevant Files\n\n")
		for _, file := range r.RelevantFiles {
			sb.WriteString(fmt.Sprintf("### `%s`\n", file.Path))
			sb.WriteString(fmt.Sprintf("%s\n", file.Relevance))
			if len(file.KeySymbols) > 0 {
				sb.WriteString(fmt.Sprintf("**Key Symbols**: %s\n", strings.Join(file.KeySymbols, ", ")))
			}
			sb.WriteString("\n")
		}
	}

	// Key Types
	if len(r.KeyTypes) > 0 {
		sb.WriteString("## Key Types & Interfaces\n\n")
		for _, typ := range r.KeyTypes {
			sb.WriteString(fmt.Sprintf("- **%s** (%s) - `%s`\n", typ.Name, typ.Kind, typ.Location))
			if len(typ.Implementations) > 0 {
				sb.WriteString(fmt.Sprintf("  - Implementations: %s\n", strings.Join(typ.Implementations, ", ")))
			}
		}
		sb.WriteString("\n")
	}

	// Dependencies
	if len(r.Dependencies) > 0 {
		sb.WriteString("## Dependencies\n\n")
		for _, dep := range r.Dependencies {
			sb.WriteString(fmt.Sprintf("- `%s` %s `%s`\n", dep.From, dep.Type, dep.To))
		}
		sb.WriteString("\n")
	}

	// Patterns
	if len(r.Patterns) > 0 {
		sb.WriteString("## Observed Patterns\n\n")
		for _, pattern := range r.Patterns {
			sb.WriteString(fmt.Sprintf("### %s\n", pattern.Name))
			sb.WriteString(fmt.Sprintf("%s\n", pattern.Description))
			if len(pattern.Examples) > 0 {
				sb.WriteString(fmt.Sprintf("**Examples**: %s\n", strings.Join(pattern.Examples, ", ")))
			}
			sb.WriteString("\n")
		}
	}

	// Risks
	if len(r.Risks) > 0 {
		sb.WriteString("## Risks & Considerations\n\n")
		for _, risk := range r.Risks {
			sb.WriteString(fmt.Sprintf("- ⚠️  %s\n", risk))
		}
		sb.WriteString("\n")
	}

	// Recommendations
	if len(r.Recommendations) > 0 {
		sb.WriteString("## Recommendations\n\n")
		for i, rec := range r.Recommendations {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
