package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"godojo/internal/core/ports"
)

// WorkspaceManager handles safe file operations within the workspace directory.
type WorkspaceManager struct {
	basePath string
}

// NewWorkspaceManager creates a WorkspaceManager rooted at basePath.
func NewWorkspaceManager(basePath string) *WorkspaceManager {
	return &WorkspaceManager{
		basePath: basePath,
	}
}

// CreateFile writes a file with the given content inside the workspace.
// Relative paths may include subdirectories. Only .go files are accepted.
// Path traversal via ".." is rejected.
func (w *WorkspaceManager) CreateFile(filename string, content string) error {
	safe, err := w.safePath(filename)
	if err != nil {
		return err
	}

	// Create parent directories
	dir := filepath.Dir(safe)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("no se pudo crear el directorio %q: %w", dir, err)
	}

	if err := os.WriteFile(safe, []byte(content), 0644); err != nil {
		return fmt.Errorf("no se pudo escribir el archivo %q: %w", filename, err)
	}

	return nil
}

// ReadFile reads the content of a file inside the workspace.
func (w *WorkspaceManager) ReadFile(filename string) (string, error) {
	safe, err := w.safePath(filename)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(safe)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("el archivo %q no existe", filename)
		}
		return "", fmt.Errorf("no se pudo leer el archivo %q: %w", filename, err)
	}

	return string(data), nil
}

// ListFiles returns a sorted list of all .go files (relative paths) in the workspace,
// including files stored inside topic subdirectories.
// Creates the base directory if it does not exist.
func (w *WorkspaceManager) ListFiles() ([]string, error) {
	// Ensure the base directory exists
	if err := os.MkdirAll(w.basePath, 0755); err != nil {
		return nil, fmt.Errorf("no se pudo crear el workspace: %w", err)
	}

	var files []string
	err := filepath.Walk(w.basePath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		rel, err := filepath.Rel(w.basePath, path)
		if err != nil {
			return fmt.Errorf("no se pudo resolver la ruta relativa de %q: %w", path, err)
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el workspace: %w", err)
	}

	sort.Strings(files)

	if files == nil {
		files = []string{}
	}

	return files, nil
}

// WorkspacePath returns the absolute path to the workspace directory.
func (w *WorkspaceManager) WorkspacePath() string {
	return w.basePath
}

// CreateDirectory creates a directory (and any parents) inside the workspace.
// Path traversal via ".." is rejected. No .go extension requirement.
func (w *WorkspaceManager) CreateDirectory(name string) error {
	safe, err := w.safeDirPath(name)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(safe, 0755); err != nil {
		return fmt.Errorf("no se pudo crear el directorio %q: %w", name, err)
	}

	return nil
}

// ListDirectory returns the entries of a directory inside the workspace,
// non-recursively (one level deep). Path traversal via ".." is rejected.
func (w *WorkspaceManager) ListDirectory(name string) ([]ports.FileInfo, error) {
	safe, err := w.safeDirPath(name)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(safe)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("el directorio %q no existe", name)
		}
		return nil, fmt.Errorf("no se pudo leer el directorio %q: %w", name, err)
	}

	result := make([]ports.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		result = append(result, ports.FileInfo{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
			Size:  info.Size(),
		})
	}

	return result, nil
}

// safeDirPath validates and resolves a relative path within the workspace
// for directory operations. Unlike safePath, it does NOT enforce .go suffix.
func (w *WorkspaceManager) safeDirPath(name string) (string, error) {
	// Reject empty name
	if name == "" {
		return "", fmt.Errorf("el nombre del directorio no puede estar vacío")
	}

	// Reject path traversal
	if strings.Contains(name, "..") {
		return "", fmt.Errorf("no se permite usar '..' en la ruta: %q", name)
	}

	// Resolve to absolute path
	abs, err := filepath.Abs(filepath.Join(w.basePath, name))
	if err != nil {
		return "", fmt.Errorf("no se pudo resolver la ruta: %w", err)
	}

	// Verify it stays within basePath
	if !strings.HasPrefix(abs, w.basePath+string(filepath.Separator)) && abs != w.basePath {
		return "", fmt.Errorf("la ruta %q está fuera del workspace", name)
	}

	return abs, nil
}

// safePath validates and resolves a relative path within the workspace.
// Returns the absolute safe path or an error.
func (w *WorkspaceManager) safePath(filename string) (string, error) {
	// Reject empty filename
	if filename == "" {
		return "", fmt.Errorf("el nombre del archivo no puede estar vacío")
	}

	// Reject path components
	if strings.Contains(filename, "..") {
		return "", fmt.Errorf("no se permite usar '..' en la ruta del archivo: %q", filename)
	}

	// Must end with .go
	if !strings.HasSuffix(filename, ".go") {
		return "", fmt.Errorf("solo se permiten archivos .go: %q", filename)
	}

	// Resolve to absolute path
	abs, err := filepath.Abs(filepath.Join(w.basePath, filename))
	if err != nil {
		return "", fmt.Errorf("no se pudo resolver la ruta: %w", err)
	}

	// Verify it stays within basePath
	if !strings.HasPrefix(abs, w.basePath+string(filepath.Separator)) && abs != w.basePath {
		return "", fmt.Errorf("la ruta %q está fuera del workspace", filename)
	}

	return abs, nil
}
