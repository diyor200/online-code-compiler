package domain

var SupportedLangs = []string{"go", "python", "javascript", "java", "cpp", "c"}

type LanguageConfig struct {
	Image        string
	CompileCmd   []string
	RunCmd       []string
	FileExt      string
	FileName     string
	NeedsCompile bool
}

var LanguageConfigs = map[string]LanguageConfig{
	"python": {
		Image:        "python:3.11-alpine",
		RunCmd:       []string{"python", "-u", "/app/main.py"},
		FileExt:      ".py",
		FileName:     "main.py",
		NeedsCompile: false,
	},
	"javascript": {
		Image:        "node:18-alpine",
		RunCmd:       []string{"node", "/app/main.js"},
		FileExt:      ".js",
		FileName:     "main.js",
		NeedsCompile: false,
	},
	"go": {
		Image:        "golang:1.21-alpine",
		CompileCmd:   []string{"go", "build", "-o", "/app/program", "/app/main.go"},
		RunCmd:       []string{"/app/program"},
		FileExt:      ".go",
		FileName:     "main.go",
		NeedsCompile: true,
	},
	"java": {
		Image:        "amazoncorretto:17-alpine",
		CompileCmd:   []string{"javac", "/app/Main.java"},
		RunCmd:       []string{"java", "-cp", "/app", "Main"},
		FileExt:      ".java",
		FileName:     "Main.java",
		NeedsCompile: true,
	},
	"cpp": {
		Image:        "gcc:13",
		CompileCmd:   []string{"g++", "-o", "/app/program", "/app/main.cpp"},
		RunCmd:       []string{"/app/program"},
		FileExt:      ".cpp",
		FileName:     "main.cpp",
		NeedsCompile: true,
	},
	"c": {
		Image:        "gcc:13",
		CompileCmd:   []string{"gcc", "-o", "/app/program", "/app/main.c"},
		RunCmd:       []string{"/app/program"},
		FileExt:      ".c",
		FileName:     "main.c",
		NeedsCompile: true,
	},
}
