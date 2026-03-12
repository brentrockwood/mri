package providers

import (
	"strings"
	"testing"
)

func TestBuildUserMessage_NoPreambleWhenNoLanguages(t *testing.T) {
	msg := buildUserMessage(PassBug, nil, nil)
	if strings.Contains(msg, repoPreamble) {
		t.Error("preamble should not appear when no languages are set")
	}
	if !strings.Contains(msg, "Perform a bug analysis") {
		t.Error("expected analysis instruction in message")
	}
}

func TestBuildUserMessage_PreambleIncludesLanguagesAndStaticText(t *testing.T) {
	langs := []string{"go", "shell"}
	msg := buildUserMessage(PassSecurity, nil, langs)

	if !strings.Contains(msg, repoPreamble) {
		t.Error("preamble constant should appear in message")
	}
	if !strings.Contains(msg, "go, shell") {
		t.Errorf("expected language list in preamble, got:\n%s", msg)
	}
	if !strings.Contains(msg, "Perform a security analysis") {
		t.Error("expected analysis instruction after preamble")
	}
	// Preamble must come before the analysis instruction.
	preambleIdx := strings.Index(msg, repoPreamble)
	analysisIdx := strings.Index(msg, "Perform a")
	if preambleIdx > analysisIdx {
		t.Error("preamble must appear before the analysis instruction")
	}
}

func TestSetAnalysisContext(t *testing.T) {
	p := &AnthropicProvider{}
	if len(p.languages) != 0 {
		t.Fatal("expected empty languages before SetAnalysisContext")
	}
	p.SetAnalysisContext([]string{"go", "python"})
	if len(p.languages) != 2 || p.languages[0] != "go" {
		t.Errorf("unexpected languages after set: %v", p.languages)
	}
}
