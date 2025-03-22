package main

import (
	"reflect"
	"testing"
)

func TestSplitWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "通常の入力",
			input:    "abc def",
			expected: []string{"abc", "def"},
		},
		{
			name:     "末尾に空白がある入力",
			input:    "abc def ",
			expected: []string{"abc", "def", ""},
		},
		{
			name:     "複数の空白を含む入力",
			input:    "  abc  def  ",
			expected: []string{"abc", "def", ""},
		},
		{
			name:     "引用符内の空白を保持",
			input:    "abc \"def ghi\" jkl",
			expected: []string{"abc", "def ghi", "jkl"},
		},
		{
			name:     "引用符と末尾の空白",
			input:    "abc \"def ghi\" ",
			expected: []string{"abc", "def ghi", ""},
		},
		{
			name:     "空の入力",
			input:    "",
			expected: []string{},
		},
		{
			name:     "空白のみの入力",
			input:    " ",
			expected: []string{""},
		},
		{
			name:     "タブを含む入力",
			input:    "abc\tdef ",
			expected: []string{"abc", "def", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitWords(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("splitWords(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
