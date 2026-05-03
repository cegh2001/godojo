package ports

// WorkspaceManager defines the contract for safe file operations
// within the student's Go workspace (~/.godojo/workspace/).
type WorkspaceManager interface {
	// CreateFile writes a file with the given content to the workspace.
	CreateFile(filename string, content string) error

	// ReadFile reads a file from the workspace by name.
	ReadFile(filename string) (string, error)

	// ListFiles returns all filenames currently in the workspace.
	ListFiles() ([]string, error)

	// WorkspacePath returns the absolute path to the workspace directory.
	WorkspacePath() string
}
