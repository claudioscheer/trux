package main

import (
	"testing"

	"github.com/claudioscheer/trux/internal/language"
)

func TestRootCommandVersionMirrorsLanguageVersion(t *testing.T) {
	if rootCmd.Version != language.Version {
		t.Fatalf("root command version = %q, want language version %q", rootCmd.Version, language.Version)
	}
}
