package ingestion

import "path/filepath"

// supported maps file extensions to canonical language names.
var supported = map[string]string{
	".go":    "go",
	".py":    "python",
	".js":    "javascript",
	".ts":    "typescript",
	".jsx":   "javascript",
	".tsx":   "typescript",
	".rb":    "ruby",
	".rs":    "rust",
	".java":  "java",
	".kt":    "kotlin",
	".swift": "swift",
	".c":     "c",
	".cpp":   "cpp",
	".cc":    "cpp",
	".cxx":   "cpp",
	".h":     "c",
	".hpp":   "cpp",
	".cs":    "csharp",
	".php":   "php",
	".scala": "scala",
	".ex":    "elixir",
	".exs":   "elixir",
	".erl":   "erlang",
	".hs":    "haskell",
	".lua":   "lua",
	".r":     "r",
	".R":     "r",
	".sh":    "shell",
	".bash":  "shell",
	".zsh":   "shell",
	".pl":    "perl",
	".pm":    "perl",
}

// DetectLanguage returns the canonical language name for the given filename,
// or an empty string if the file extension is not recognized as a code file.
func DetectLanguage(filename string) string {
	ext := filepath.Ext(filename)
	return supported[ext]
}
