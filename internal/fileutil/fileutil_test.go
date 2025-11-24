package fileutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special chars",
			input:    "photo.jpg",
			expected: "photo",
		},
		{
			name:     "with colon",
			input:    "photo:test.jpg",
			expected: "photo_test",
		},
		{
			name:     "with multiple invalid chars",
			input:    "photo*test?file.jpg",
			expected: "photo_test_file",
		},
		{
			name:     "with quotes",
			input:    "photo\"test.jpg",
			expected: "photo_test",
		},
		{
			name:     "with angle brackets",
			input:    "photo<test>.jpg",
			expected: "photo_test_",
		},
		{
			name:     "with pipe",
			input:    "photo|test.jpg",
			expected: "photo_test",
		},
		{
			name:     "with semicolon",
			input:    "photo;test.jpg",
			expected: "photo_test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetOutputFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		format   string
		expected string
	}{
		{
			name:     "jpg to webp",
			input:    "photo.jpg",
			format:   "webp",
			expected: "photo.webp",
		},
		{
			name:     "jpg to jpg",
			input:    "photo.jpg",
			format:   "jpg",
			expected: "photo.jpg",
		},
		{
			name:     "heic to webp",
			input:    "photo.heic",
			format:   "webp",
			expected: "photo.webp",
		},
		{
			name:     "with special chars to webp",
			input:    "photo:test.jpg",
			format:   "webp",
			expected: "photo_test.webp",
		},
		{
			name:     "with special chars to jpeg",
			input:    "photo*test?.jpg",
			format:   "jpeg",
			expected: "photo_test_.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOutputFilename(tt.input, tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}
