package codebeacon

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// ExtractReportFromHistory attempts to extract a JSON BeaconReport from the agent's conversation history.
// It looks for JSON in assistant messages (primary) or tool messages (fallback).
// It also populates the FilesRead field by scanning tool calls in the history.
func ExtractReportFromHistory(history []engine.ChatMessage) (*BeaconReport, error) {
	// Search backwards through history to find the most recent report
	var report *BeaconReport
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]

		// Priority 1: Check assistant messages for JSON (most common location)
		// The agent outputs JSON in an assistant message, then calls respond
		if msg.Role == engine.RoleAssistant && msg.Content != "" {
			if r, err := extractJSONFromContent(msg.Content); err == nil {
				report = r
				break
			}
		}

		// Priority 2: Check tool messages from 'respond' tool (legacy/fallback)
		if msg.Role == engine.RoleTool && msg.Name != "" {
			// The respond tool result is in the format: {"status":"complete","summary":"..."}
			// We need to extract the summary field first, then parse it as JSON
			if r, err := extractFromRespondToolResult(msg.Content); err == nil {
				report = r
				break
			}
			// Fallback: try parsing the entire content as JSON
			if r, err := extractJSONFromContent(msg.Content); err == nil {
				report = r
				break
			}
		}
	}

	if report == nil {
		return nil, fmt.Errorf("no valid BeaconReport found in history")
	}

	// Populate FilesRead by scanning tool calls
	report.FilesRead = extractFilesReadFromHistory(history)

	return report, nil
}

// extractFilesReadFromHistory scans the conversation history to find all files
// that were read by CodeBeacon (via read_file or read_span tools).
func extractFilesReadFromHistory(history []engine.ChatMessage) []string {
	filesMap := make(map[string]bool)
	var files []string

	for _, msg := range history {
		// Look for tool calls in assistant messages
		if msg.Role == engine.RoleAssistant {
			for _, tc := range msg.ToolCalls {
				if tc.Name == "read_file" || tc.Name == "read_span" {
					// Extract path from tool arguments
					if path, ok := tc.Args["path"].(string); ok && path != "" {
						if !filesMap[path] {
							filesMap[path] = true
							files = append(files, path)
						}
					}
					// Also check for "target_file" (read_file uses this)
					if path, ok := tc.Args["target_file"].(string); ok && path != "" {
						if !filesMap[path] {
							filesMap[path] = true
							files = append(files, path)
						}
					}
				}
			}
		}
	}

	return files
}

// extractFromRespondToolResult extracts the BeaconReport from a respond tool's result.
// The respond tool returns: {"status":"complete","summary":"<JSON_STRING>"}
// We need to parse the outer JSON, extract the summary field, then parse that as BeaconReport.
func extractFromRespondToolResult(content string) (*BeaconReport, error) {
	// Try to parse as respond tool result
	var respondResult struct {
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}

	if err := json.Unmarshal([]byte(content), &respondResult); err != nil {
		return nil, fmt.Errorf("not a respond tool result: %w", err)
	}

	if respondResult.Summary == "" {
		return nil, fmt.Errorf("respond tool result has empty summary")
	}

	// Now try to parse the summary as a BeaconReport
	return extractJSONFromContent(respondResult.Summary)
}

// extractJSONFromContent attempts to extract a BeaconReport from text content.
// It looks for:
// 1. JSON code blocks (```json ... ```)
// 2. Plain JSON objects starting with {
func extractJSONFromContent(content string) (*BeaconReport, error) {
	// Try to find JSON code block
	jsonBlockRegex := regexp.MustCompile("```json\\s*\\n([\\s\\S]*?)\\n```")
	matches := jsonBlockRegex.FindStringSubmatch(content)

	var jsonText string
	if len(matches) > 1 {
		jsonText = strings.TrimSpace(matches[1])
	} else {
		if extracted, ok := extractFirstJSONObject(content); ok {
			jsonText = extracted
		}
	}

	if jsonText == "" {
		return nil, fmt.Errorf("no JSON content found")
	}

	// Attempt to parse as BeaconReport
	var report BeaconReport
	if err := json.Unmarshal([]byte(jsonText), &report); err != nil {
		return nil, fmt.Errorf("failed to parse JSON as BeaconReport: %w", err)
	}

	// Validate required fields
	if report.InvestigationGoal == "" {
		return nil, fmt.Errorf("report missing investigation_goal field")
	}

	return &report, nil
}

// extractFirstJSONObject scans text to find the first JSON object (starting at '{').
// It tolerates leading labels such as "summary:" or other prose before the JSON payload.
func extractFirstJSONObject(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false
	}

	start := strings.Index(trimmed, "{")
	if start == -1 {
		return "", false
	}

	decoder := json.NewDecoder(strings.NewReader(trimmed[start:]))
	decoder.UseNumber()

	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return "", false
	}

	return strings.TrimSpace(string(raw)), true
}

// CreateFallbackReport creates a minimal report when JSON parsing fails.
// This ensures CodeBeacon always returns something useful even if formatting is wrong.
func CreateFallbackReport(goal string, history []engine.ChatMessage) *BeaconReport {
	// Extract text from assistant messages
	var findings []string
	for _, msg := range history {
		if msg.Role == engine.RoleAssistant && msg.Content != "" {
			findings = append(findings, msg.Content)
		}
	}

	summary := "Investigation completed but failed to produce structured JSON report. Raw findings:\n\n"
	summary += strings.Join(findings, "\n\n")

	// Extract files that were read even in fallback mode
	filesRead := extractFilesReadFromHistory(history)

	return &BeaconReport{
		InvestigationGoal: goal,
		Summary:           summary,
		RelevantFiles:     []FileAnalysis{},
		KeyTypes:          []TypeInfo{},
		Dependencies:      []Dependency{},
		Patterns:          []Pattern{},
		Risks:             []string{"Report parsing failed - review raw output carefully"},
		Recommendations:   []string{"Manually review investigation findings above"},
		FilesRead:         filesRead,
	}
}
