package ingestion

import "testing"

func TestDetectLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{name: "go", filename: "main.go", want: "go"},
		{name: "python", filename: "app.py", want: "python"},
		{name: "javascript", filename: "index.js", want: "javascript"},
		{name: "jsx", filename: "App.jsx", want: "javascript"},
		{name: "typescript", filename: "server.ts", want: "typescript"},
		{name: "tsx", filename: "Component.tsx", want: "typescript"},
		{name: "ruby", filename: "Gemfile.rb", want: "ruby"},
		{name: "rust", filename: "lib.rs", want: "rust"},
		{name: "java", filename: "Main.java", want: "java"},
		{name: "kotlin", filename: "App.kt", want: "kotlin"},
		{name: "swift", filename: "App.swift", want: "swift"},
		{name: "c", filename: "main.c", want: "c"},
		{name: "c header", filename: "types.h", want: "c"},
		{name: "cpp", filename: "engine.cpp", want: "cpp"},
		{name: "cpp cc", filename: "engine.cc", want: "cpp"},
		{name: "cpp cxx", filename: "engine.cxx", want: "cpp"},
		{name: "cpp hpp", filename: "engine.hpp", want: "cpp"},
		{name: "csharp", filename: "Program.cs", want: "csharp"},
		{name: "php", filename: "index.php", want: "php"},
		{name: "scala", filename: "Main.scala", want: "scala"},
		{name: "elixir", filename: "app.ex", want: "elixir"},
		{name: "elixir script", filename: "mix.exs", want: "elixir"},
		{name: "erlang", filename: "server.erl", want: "erlang"},
		{name: "haskell", filename: "Main.hs", want: "haskell"},
		{name: "lua", filename: "script.lua", want: "lua"},
		{name: "r lowercase", filename: "analysis.r", want: "r"},
		{name: "r uppercase", filename: "analysis.R", want: "r"},
		{name: "shell sh", filename: "run.sh", want: "shell"},
		{name: "shell bash", filename: "setup.bash", want: "shell"},
		{name: "shell zsh", filename: "init.zsh", want: "shell"},
		{name: "perl", filename: "script.pl", want: "perl"},
		{name: "perl module", filename: "module.pm", want: "perl"},
		// Non-code files should return empty string.
		{name: "markdown", filename: "README.md", want: ""},
		{name: "yaml", filename: "config.yaml", want: ""},
		{name: "json", filename: "package.json", want: ""},
		{name: "txt", filename: "notes.txt", want: ""},
		{name: "no extension", filename: "Makefile", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := DetectLanguage(tc.filename)
			if got != tc.want {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tc.filename, got, tc.want)
			}
		})
	}
}
