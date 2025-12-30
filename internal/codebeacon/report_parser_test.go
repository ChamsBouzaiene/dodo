package codebeacon

import (
	"testing"
)

func TestExtractJSONFromContent_AllowsCodeBlock(t *testing.T) {
	content := "Here is the report:\n```json\n{\n  \"investigation_goal\": \"Test\",\n  \"summary\": \"works\"\n}\n```"
	report, err := extractJSONFromContent(content)
	if err != nil {
		t.Fatalf("expected parse to succeed, got error: %v", err)
	}
	if report.InvestigationGoal != "Test" {
		t.Fatalf("expected investigation_goal to be parsed, got %q", report.InvestigationGoal)
	}
}

func TestExtractJSONFromContent_IgnoresPrefix(t *testing.T) {
	content := "summary:{\"investigation_goal\":\"Test\",\"summary\":\"works\"}"
	report, err := extractJSONFromContent(content)
	if err != nil {
		t.Fatalf("expected parse to succeed, got error: %v", err)
	}
	if report.Summary != "works" {
		t.Fatalf("expected summary to match, got %q", report.Summary)
	}
}
