package ports_test

import (
	"testing"

	"godojo/internal/core/ports"
)

func TestFileInfo_StructFields(t *testing.T) {
	tests := []struct {
		name     string
		fi       ports.FileInfo
		wantName string
		wantDir  bool
		wantSize int64
	}{
		{
			name: "regular file with content",
			fi: ports.FileInfo{
				Name:  "main.go",
				IsDir: false,
				Size:  1024,
			},
			wantName: "main.go",
			wantDir:  false,
			wantSize: 1024,
		},
		{
			name: "directory entry",
			fi: ports.FileInfo{
				Name:  "variables",
				IsDir: true,
				Size:  0,
			},
			wantName: "variables",
			wantDir:  true,
			wantSize: 0,
		},
		{
			name: "empty file with zero size",
			fi: ports.FileInfo{
				Name:  "empty.txt",
				IsDir: false,
				Size:  0,
			},
			wantName: "empty.txt",
			wantDir:  false,
			wantSize: 0,
		},
		{
			name: "zero value defaults",
			fi:   ports.FileInfo{},
			wantName: "",
			wantDir:  false,
			wantSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fi.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", tt.fi.Name, tt.wantName)
			}
			if tt.fi.IsDir != tt.wantDir {
				t.Errorf("IsDir = %v, want %v", tt.fi.IsDir, tt.wantDir)
			}
			if tt.fi.Size != tt.wantSize {
				t.Errorf("Size = %d, want %d", tt.fi.Size, tt.wantSize)
			}
		})
	}
}
