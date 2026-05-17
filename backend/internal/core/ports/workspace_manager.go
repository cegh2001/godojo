package ports

// FileInfo represents metadata about a file or directory entry
// returned by WorkspaceManager.ListDirectory.
type FileInfo struct {
	Name  string
	IsDir bool
	Size  int64
}

// WorkspaceManager defines the contract for safe file operations
// within the student's Go workspace (~/.godojo/workspace/).
type WorkspaceManager interface {
	// CreateFile writes a file with the given content to the workspace.
	// The path can be relative and include subdirectories.
	CreateFile(filename string, content string) error

	// ReadFile reads a file from the workspace by name.
	ReadFile(filename string) (string, error)

	// ListFiles returns all filenames currently in the workspace.
	ListFiles() ([]string, error)

	// WorkspacePath returns the absolute path to the workspace directory.
	WorkspacePath() string

	// CreateDirectory creates a directory (and any parents) inside the workspace.
	// Path traversal via ".." is rejected.
	CreateDirectory(name string) error

	// ListDirectory returns the entries of a directory inside the workspace,
	// non-recursively (one level deep). Path traversal is rejected.
	ListDirectory(name string) ([]FileInfo, error)
}
