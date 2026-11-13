package main

import (
	"strings"
	"testing"
)

// TestExtractProgramsJoinsMarkedFragments verifies the checked program comes from the actual Markdown. TestExtractProgramsJoinsMarkedFragments 验证受检程序由实际 Markdown 片段组成。
func TestExtractProgramsJoinsMarkedFragments(t *testing.T) {
	block := "<!-- dtoken-compile: quick-start -->\r\n```go\r\n%s\r\n```\r\n"
	text := strings.Replace(block, "%s", "package main", 1) + "```go\nignored()\n```\n" + strings.Replace(block, "%s", "func main() {}", 1)
	programs, err := extractPrograms(text)
	if err != nil || len(programs) != 1 || programs["quick-start"] != "package main\nfunc main() {}\n" {
		t.Fatalf("programs = %v, %v", programs, err)
	}
	for _, malformed := range []string{"", "<!-- dtoken-compile: sample -->", text + "<!-- dtoken-compile: missing -->"} {
		if _, err := extractPrograms(malformed); err == nil {
			t.Fatal("missing snippet passed validation")
		}
	}
}
