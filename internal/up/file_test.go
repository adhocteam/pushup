package up

import (
	"os"
	"path/filepath"
	"testing"
)

// Helper function to create a temporary file structure
func createTestFiles(baseDir string, files []string) error {
	for _, file := range files {
		fullPath := filepath.Join(baseDir, file)
		if err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err != nil {
			return err
		}
		_, err := os.Create(fullPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestFind(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Define the file structure
	files := []string{
		"example.up.go",
		"test_up.go",
		"main.up.go",
		"script.up",
		"readme.md",
		"nested/example.up",
		"nested/example.go",
	}

	if err := createTestFiles(tempDir, files); err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}

	tests := []struct {
		name     string
		root     string
		fileType string
		expected []string
	}{
		{
			name:     "Find .up.go files",
			root:     tempDir,
			fileType: "go",
			expected: []string{
				filepath.Join(tempDir, "example.up.go"),
				filepath.Join(tempDir, "main.up.go"),
			},
		},
		{
			name:     "Find .up files",
			root:     tempDir,
			fileType: "up",
			expected: []string{
				filepath.Join(tempDir, "script.up"),
				filepath.Join(tempDir, "nested/example.up"),
			},
		},
		{
			name:     "No match for .txt files",
			root:     tempDir,
			fileType: "txt",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var results []string
			findFunc := Find(tt.root, tt.fileType)
			findFunc(func(path string) bool {
				results = append(results, path)
				return true
			})

			if len(results) != len(tt.expected) {
				t.Errorf("Expected %d results, got %d", len(tt.expected), len(results))
			}

			for _, expectedPath := range tt.expected {
				found := false
				for _, resultPath := range results {
					if expectedPath == resultPath {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected path %s not found in results", expectedPath)
				}
			}
		})
	}
}
