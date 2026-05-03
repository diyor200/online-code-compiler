package domain

var SupportedLangs = []string{"go", "python", "javascript", "java", "cpp", "c"}

type LanguageConfig struct {
	Image        string
	CompileCmd   []string
	RunCmd       []string
	NeedsCompile bool
	StdinCompile bool // true = pipe code via stdin to compiler (C, C++)
	SourceFile   string
}

var LanguageConfigs = map[string]LanguageConfig{
	"python": {
		Image:        "python:3.11-alpine",
		RunCmd:       []string{"python", "-u", "-c", ""}, // code injected at runtime
		NeedsCompile: false,
		StdinCompile: false,
	},
	"javascript": {
		Image:        "node:18-alpine",
		RunCmd:       []string{"node", "-e", ""}, // code injected at runtime
		NeedsCompile: false,
		StdinCompile: false,
	},
	"c": {
		Image:        "gcc:13",
		CompileCmd:   []string{"gcc", "-x", "c", "-o", "/app/program", "-"},
		RunCmd:       []string{"/app/program"},
		NeedsCompile: true,
		StdinCompile: true, // code piped via stdin
	},
	"cpp": {
		Image:        "gcc:13",
		CompileCmd:   []string{"g++", "-x", "c++", "-o", "/app/program", "-"},
		RunCmd:       []string{"/app/program"},
		NeedsCompile: true,
		StdinCompile: true, // code piped via stdin
	},
	"go": {
		Image:        "golang:1.22-alpine",
		CompileCmd:   []string{"go", "build", "-o", "/app/program", "/app/main.go"},
		RunCmd:       []string{"/app/program"},
		NeedsCompile: true,
		StdinCompile: false, // tmpfs mount, code written to file
		SourceFile:   "main.go",
	},
	"java": {
		Image:        "amazoncorretto:17-alpine",
		CompileCmd:   []string{"javac", "/app/Main.java"},
		RunCmd:       []string{"java", "-cp", "/app", "Main"},
		NeedsCompile: true,
		StdinCompile: false, // tmpfs mount, code written to file
		SourceFile:   "Main.java",
	},
}
