package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"godojo/internal/adapters/workspace"
)

func TestCreateFile_ValidGoFile(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	err := wm.CreateFile("hola-mundo.go", "package main\n\nfunc main() {}")
	if err != nil {
		t.Fatalf("CreateFile returned unexpected error: %v", err)
	}

	// Verify file exists on disk
	content, err := os.ReadFile(filepath.Join(dir, "hola-mundo.go"))
	if err != nil {
		t.Fatalf("file not created on disk: %v", err)
	}
	if string(content) != "package main\n\nfunc main() {}" {
		t.Errorf("file content = %q, want %q", string(content), "package main\n\nfunc main() {}")
	}
}

func TestCreateFile_PathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	err := wm.CreateFile("../../etc/passwd", "malicious")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

func TestCreateFile_DotDotAnywhereRejected(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	attacks := []string{
		"../etc/passwd.go",
		"sub/../../passwd.go",
		"..",
		"a/../b/test.go",
	}

	for _, filename := range attacks {
		t.Run("attack="+filename, func(t *testing.T) {
			err := wm.CreateFile(filename, "malicious")
			if err == nil {
				t.Errorf("expected error for %q, got nil", filename)
			}
		})
	}
}

func TestCreateFile_NonGoExtensionRejected(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	err := wm.CreateFile("script.py", "print('hello')")
	if err == nil {
		t.Fatal("expected error for non-.go file, got nil")
	}

	err = wm.CreateFile("README.md", "# docs")
	if err == nil {
		t.Fatal("expected error for non-.go file, got nil")
	}

	err = wm.CreateFile("file", "no extension")
	if err == nil {
		t.Fatal("expected error for file without .go extension, got nil")
	}
}

func TestCreateFile_Subdirectory(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	err := wm.CreateFile("subdir/variables.go", "package main\n\nvar x int")
	if err != nil {
		t.Fatalf("CreateFile in subdir returned unexpected error: %v", err)
	}

	// Verify file exists in subdirectory
	content, err := os.ReadFile(filepath.Join(dir, "subdir", "variables.go"))
	if err != nil {
		t.Fatalf("file not created in subdir: %v", err)
	}
	if !strings.Contains(string(content), "var x int") {
		t.Errorf("file content missing expected text. Got: %q", string(content))
	}
}

func TestReadFile_Valid(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	// Create a file first
	if err := wm.CreateFile("test.go", "package main"); err != nil {
		t.Fatalf("setup: CreateFile failed: %v", err)
	}

	content, err := wm.ReadFile("test.go")
	if err != nil {
		t.Fatalf("ReadFile returned unexpected error: %v", err)
	}
	if content != "package main" {
		t.Errorf("content = %q, want 'package main'", content)
	}
}

func TestReadFile_NonExistent(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	_, err := wm.ReadFile("no_existe.go")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestListFiles_ReturnsSortedAndCreatesDir(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "workspace")
	wm := workspace.NewWorkspaceManager(base)

	// ListFiles should create the base directory if it doesn't exist,
	// and return empty list
	files, err := wm.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles returned unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files in empty workspace, got %d", len(files))
	}

	// Verify the directory was created
	if _, err := os.Stat(base); os.IsNotExist(err) {
		t.Errorf("workspace directory was not created")
	}

	// Create some .go files
	wm.CreateFile("z.go", "package main")
	wm.CreateFile("a.go", "package main")
	wm.CreateFile("m.go", "package main")
	// Also create a non-.go file directly (should be ignored by ListFiles)
	os.WriteFile(filepath.Join(base, "not_a_go_file.txt"), []byte("ignored"), 0644)

	files, err = wm.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles after creation returned unexpected error: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d: %v", len(files), files)
	}

	// Must be sorted
	expected := []string{"a.go", "m.go", "z.go"}
	for i, f := range files {
		if f != expected[i] {
			t.Errorf("files[%d] = %q, want %q (not sorted?)", i, f, expected[i])
		}
	}
}

func TestListFiles_IncludesNestedGoFiles(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	if err := wm.CreateFile("variables/clase-1.go", "package main"); err != nil {
		t.Fatalf("CreateFile nested failed: %v", err)
	}
	if err := wm.CreateFile("maps/ejercicio.go", "package main"); err != nil {
		t.Fatalf("CreateFile nested failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "variables", "notas.txt"), []byte("ignorar"), 0644); err != nil {
		t.Fatalf("WriteFile nested txt failed: %v", err)
	}

	files, err := wm.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles() error: %v", err)
	}

	expected := []string{"maps/ejercicio.go", "variables/clase-1.go"}
	if len(files) != len(expected) {
		t.Fatalf("expected %d files, got %d: %v", len(expected), len(files), files)
	}
	for i := range expected {
		if files[i] != expected[i] {
			t.Errorf("files[%d] = %q, want %q", i, files[i], expected[i])
		}
	}
}

func TestWorkspacePath(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	path := wm.WorkspacePath()
	if path != dir {
		t.Errorf("WorkspacePath() = %q, want %q", path, dir)
	}
}

func TestCreateFile_EmptyFilenameRejected(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	err := wm.CreateFile("", "content")
	if err == nil {
		t.Fatal("expected error for empty filename, got nil")
	}
}

func TestCreateFile_EmptyContent(t *testing.T) {
	// Empty content should be valid — creates an empty .go file
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	err := wm.CreateFile("vacio.go", "")
	if err != nil {
		t.Fatalf("CreateFile with empty content should succeed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "vacio.go"))
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("expected empty file, got %d bytes: %q", len(content), string(content))
	}
}

func TestCreateFile_NoExtensionRejected(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	err := wm.CreateFile("sin_extension", "content")
	if err == nil {
		t.Fatal("expected error for file without extension, got nil")
	}
}

// --- CreateDirectory tests ---

func TestCreateDirectory_Valid(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	err := wm.CreateDirectory("variables")
	if err != nil {
		t.Fatalf("CreateDirectory returned unexpected error: %v", err)
	}

	// Verify directory exists on disk
	fi, err := os.Stat(filepath.Join(dir, "variables"))
	if err != nil {
		t.Fatalf("directory not created on disk: %v", err)
	}
	if !fi.IsDir() {
		t.Error("created path is not a directory")
	}
}

func TestCreateDirectory_Nested(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	err := wm.CreateDirectory("variables/ejercicios/nivel-1")
	if err != nil {
		t.Fatalf("CreateDirectory nested returned unexpected error: %v", err)
	}

	// Verify whole tree exists
	nested := filepath.Join(dir, "variables", "ejercicios", "nivel-1")
	fi, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("nested directory not created: %v", err)
	}
	if !fi.IsDir() {
		t.Error("nested created path is not a directory")
	}
}

func TestCreateDirectory_PathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	err := wm.CreateDirectory("../../etc")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
}

func TestCreateDirectory_DotDotAnywhereRejected(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	attacks := []string{
		"../escape",
		"sub/../../passwd",
		"..",
		"a/../b",
	}

	for _, name := range attacks {
		t.Run("attack="+name, func(t *testing.T) {
			err := wm.CreateDirectory(name)
			if err == nil {
				t.Errorf("expected error for %q, got nil", name)
			}
		})
	}
}

func TestCreateDirectory_EmptyNameRejected(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	err := wm.CreateDirectory("")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestCreateDirectory_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	// Create the directory first directly
	existingDir := filepath.Join(dir, "existing")
	if err := os.MkdirAll(existingDir, 0755); err != nil {
		t.Fatalf("setup MkdirAll failed: %v", err)
	}

	// CreateDirectory should succeed (MkdirAll is idempotent)
	err := wm.CreateDirectory("existing")
	if err != nil {
		t.Errorf("CreateDirectory on existing dir should succeed: %v", err)
	}
}

// --- ListDirectory tests ---

func TestListDirectory_ReturnsEntries(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	// Populate directory with files and subdirectories
	if err := wm.CreateFile("main.go", "package main"); err != nil {
		t.Fatalf("setup CreateFile failed: %v", err)
	}
	if err := wm.CreateFile("test.go", "package main"); err != nil {
		t.Fatalf("setup CreateFile failed: %v", err)
	}
	if err := wm.CreateDirectory("subdir"); err != nil {
		t.Fatalf("setup CreateDirectory failed: %v", err)
	}

	entries, err := wm.ListDirectory(".")
	if err != nil {
		t.Fatalf("ListDirectory returned unexpected error: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d: %+v", len(entries), entries)
	}

	found := map[string]bool{}
	for _, e := range entries {
		found[e.Name] = true
	}

	if !found["main.go"] {
		t.Error("missing main.go in listing")
	}
	if !found["test.go"] {
		t.Error("missing test.go in listing")
	}
	if !found["subdir"] {
		t.Error("missing subdir in listing")
	}

	// Verify subdir is marked as directory
	for _, e := range entries {
		if e.Name == "subdir" && !e.IsDir {
			t.Error("subdir should have IsDir=true")
		}
	}
}

func TestListDirectory_NonRecursive(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	if err := wm.CreateFile("top.go", "package main"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := wm.CreateDirectory("sub"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := wm.CreateFile("sub/nested.go", "package main"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	entries, err := wm.ListDirectory(".")
	if err != nil {
		t.Fatalf("ListDirectory(.): %v", err)
	}

	// Should only see top.go and sub (not sub/nested.go)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries at root, got %d: %+v", len(entries), entries)
	}

	for _, e := range entries {
		if e.Name == "nested.go" {
			t.Error("ListDirectory should be non-recursive, found nested.go at root")
		}
	}

	// Now list the sub directory
	subEntries, err := wm.ListDirectory("sub")
	if err != nil {
		t.Fatalf("ListDirectory(sub): %v", err)
	}
	if len(subEntries) != 1 {
		t.Fatalf("expected 1 entry in sub, got %d", len(subEntries))
	}
	if subEntries[0].Name != "nested.go" {
		t.Errorf("expected nested.go in sub, got %q", subEntries[0].Name)
	}
}

func TestListDirectory_FileSize(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	if err := wm.CreateFile("sized.go", "package main\n\nfunc main() {}"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	entries, err := wm.ListDirectory(".")
	if err != nil {
		t.Fatalf("ListDirectory: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Size <= 0 {
		t.Errorf("file size should be > 0, got %d", entries[0].Size)
	}
	if entries[0].IsDir {
		t.Error("file should have IsDir=false")
	}
}

func TestListDirectory_MissingDir(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	_, err := wm.ListDirectory("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent directory, got nil")
	}
}

func TestListDirectory_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	// Create an empty directory
	if err := wm.CreateDirectory("empty"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	entries, err := wm.ListDirectory("empty")
	if err != nil {
		t.Fatalf("ListDirectory empty dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries in empty dir, got %d", len(entries))
	}
}

func TestListDirectory_PathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	_, err := wm.ListDirectory("../etc")
	if err == nil {
		t.Fatal("expected error for path traversal in ListDirectory, got nil")
	}
}

func TestListDirectory_Subdirectory(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	if err := wm.CreateDirectory("variables"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := wm.CreateFile("variables/main.go", "package main"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := wm.CreateFile("variables/main_test.go", "package main"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	entries, err := wm.ListDirectory("variables")
	if err != nil {
		t.Fatalf("ListDirectory(variables): %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	names := []string{entries[0].Name, entries[1].Name}
	if names[0] != "main.go" {
		t.Errorf("first entry should be main.go, got %q", names[0])
	}
	if names[1] != "main_test.go" {
		t.Errorf("second entry should be main_test.go, got %q", names[1])
	}
}

func TestWriteThenRead_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	wm := workspace.NewWorkspaceManager(dir)

	original := `package main

import "fmt"

func main() {
	fmt.Println("Round-trip test")
}
`
	err := wm.CreateFile("roundtrip.go", original)
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}

	read, err := wm.ReadFile("roundtrip.go")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if read != original {
		t.Errorf("round-trip mismatch.\nWrote: %q\nRead:  %q", original, read)
	}
}
