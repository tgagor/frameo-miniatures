package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIgnoreMatcher(t *testing.T) {
	// Setup temporary directory
	tmpDir, err := os.MkdirTemp("", "frameo-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create .frameoignore
	ignoreContent := `
*/ignored_subdir/*
*.bad
*/2005.07/Ognisko u Gogusia/*
2002.03/Studniówka z Beatą/*
`
	err = os.WriteFile(filepath.Join(tmpDir, ".frameoignore"), []byte(ignoreContent), 0644)
	require.NoError(t, err)

	// Initialize matcher
	matcher, err := NewIgnoreMatcher("", tmpDir)
	require.NoError(t, err)

	tests := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{"normal.jpg", false, false},
		{"file.bad", false, true},
		{"sub/file.bad", false, true},
		{"ignored_subdir/file.jpg", false, false}, // pattern is */ignored_subdir/*, so relative path "ignored_subdir/..." shouldn't match?
		// Wait, if pattern is */foo/*, it expects a parent.
		// If I have "root/ignored_subdir/file.jpg", relative is "ignored_subdir/file.jpg".
		// "ignored_subdir" does NOT match "*/ignored_subdir".

		// Let's test the user's scenario
		// User has: */2005.07/...
		// File: original/2005.07/...
		// If we run with -i original/, relative path is 2005.07/...
		// But we added a check for FULL path too.
		// Full path: original/2005.07/...
		// This SHOULD match */2005.07/...
		// This SHOULD match */2005.07/... IF we provide a parent directory (simulating full path check)
		{"original/2005.07/Ognisko u Gogusia/", true, true},
		{"original/2005.07/Ognisko u Gogusia/Ognisko u Gogusia 08.JPG", false, true},
		// 2002.03 pattern does not have leading */, so it matches relative path directly
		{"2002.03/Studniówka z Beatą/", true, true},
		{"2002.03/Studniówka z Beatą/Zdięcia studniówka 2002/ZDJ4.JPG", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, matcher.Matches(tt.path, tt.isDir))
		})
	}
}

func TestWalkFiles_UserScenario(t *testing.T) {
	// Simulate the user's exact scenario
	// root/
	//   .frameoignore: */subdir/*
	//   subdir/
	//     image.jpg

	tmpDir, err := os.MkdirTemp("", "frameo-user-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create .frameoignore
	ignoreContent := `*/subdir/*`
	err = os.WriteFile(filepath.Join(tmpDir, ".frameoignore"), []byte(ignoreContent), 0644)
	require.NoError(t, err)

	// Create structure: root/original/subdir/image.jpg
	// The user ran: -i original/
	// So the .frameoignore is likely in original/? Or in the current dir?
	// "Look for a .frameoignore file in the root of the input directory." -> SPEC 3.1

	// Scenario A: .frameoignore is in input dir
	inputDir := filepath.Join(tmpDir, "original")
	err = os.MkdirAll(filepath.Join(inputDir, "subdir"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(inputDir, ".frameoignore"), []byte(ignoreContent), 0644)
	require.NoError(t, err)

	imagePath := filepath.Join(inputDir, "subdir", "image.jpg")
	err = os.WriteFile(imagePath, []byte("fake image"), 0644)
	require.NoError(t, err)

	// Matcher
	matcher, err := NewIgnoreMatcher("", inputDir)
	require.NoError(t, err)

	// Walk
	files := make(chan File, 10)
	go WalkFiles(inputDir, files, matcher)

	found := false
	for f := range files {
		if f.Path == imagePath {
			found = true
		}
	}

	// Should NOT be found
	assert.False(t, found, "File should have been ignored")
}
