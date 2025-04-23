package utils

import (
	"reflect"
	"testing"
)

func TestUint32ToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    uint32
		size     int
		expected []byte
	}{
		{
			name:     "1バイト変換 (0)",
			input:    0,
			size:     1,
			expected: []byte{0x00},
		},
		{
			name:     "1バイト変換 (255)",
			input:    255,
			size:     1,
			expected: []byte{0xFF},
		},
		{
			name:     "2バイト変換 (256)",
			input:    256,
			size:     2,
			expected: []byte{0x01, 0x00},
		},
		{
			name:     "2バイト変換 (65535)",
			input:    65535,
			size:     2,
			expected: []byte{0xFF, 0xFF},
		},
		{
			name:     "3バイト変換 (65536)",
			input:    65536,
			size:     3,
			expected: []byte{0x01, 0x00, 0x00},
		},
		{
			name:     "3バイト変換 (16777215)",
			input:    16777215,
			size:     3,
			expected: []byte{0xFF, 0xFF, 0xFF},
		},
		{
			name:     "4バイト変換 (16777216)",
			input:    16777216,
			size:     4,
			expected: []byte{0x01, 0x00, 0x00, 0x00},
		},
		{
			name:     "4バイト変換 (4294967295)",
			input:    4294967295,
			size:     4,
			expected: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Uint32ToBytes(tt.input, tt.size)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Uint32ToBytes(%d, %d) = %v, want %v", tt.input, tt.size, result, tt.expected)
			}
		})
	}

	// パニックテスト
	t.Run("サイズが範囲外の場合はパニック", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Uint32ToBytes(0, 0) did not panic")
			}
		}()
		Uint32ToBytes(0, 0)
	})

	t.Run("サイズが範囲外の場合はパニック", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Uint32ToBytes(0, 5) did not panic")
			}
		}()
		Uint32ToBytes(0, 5)
	})
}

func TestBytesToUint32(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected uint32
	}{
		{
			name:     "1バイト変換 (0)",
			input:    []byte{0x00},
			expected: 0,
		},
		{
			name:     "1バイト変換 (255)",
			input:    []byte{0xFF},
			expected: 255,
		},
		{
			name:     "2バイト変換 (256)",
			input:    []byte{0x01, 0x00},
			expected: 256,
		},
		{
			name:     "2バイト変換 (65535)",
			input:    []byte{0xFF, 0xFF},
			expected: 65535,
		},
		{
			name:     "3バイト変換 (65536)",
			input:    []byte{0x01, 0x00, 0x00},
			expected: 65536,
		},
		{
			name:     "3バイト変換 (16777215)",
			input:    []byte{0xFF, 0xFF, 0xFF},
			expected: 16777215,
		},
		{
			name:     "4バイト変換 (16777216)",
			input:    []byte{0x01, 0x00, 0x00, 0x00},
			expected: 16777216,
		},
		{
			name:     "4バイト変換 (4294967295)",
			input:    []byte{0xFF, 0xFF, 0xFF, 0xFF},
			expected: 4294967295,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BytesToUint32(tt.input)
			if result != tt.expected {
				t.Errorf("BytesToUint32(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}

	// パニックテスト
	t.Run("空のスライスの場合はパニック", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("BytesToUint32([]byte{}) did not panic")
			}
		}()
		BytesToUint32([]byte{})
	})

	t.Run("5バイト以上のスライスの場合はパニック", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("BytesToUint32([]byte{0,0,0,0,0}) did not panic")
			}
		}()
		BytesToUint32([]byte{0, 0, 0, 0, 0})
	})
}

func TestBytesToInt32(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected int32
	}{
		{
			name:     "1バイト変換 (0)",
			input:    []byte{0x00},
			expected: 0,
		},
		{
			name:     "1バイト変換 (127)",
			input:    []byte{0x7F},
			expected: 127,
		},
		{
			name:     "1バイト変換 (-128)",
			input:    []byte{0x80},
			expected: -128,
		},
		{
			name:     "1バイト変換 (-1)",
			input:    []byte{0xFF},
			expected: -1,
		},
		{
			name:     "2バイト変換 (128)",
			input:    []byte{0x00, 0x80},
			expected: 128,
		},
		{
			name:     "2バイト変換 (32767)",
			input:    []byte{0x7F, 0xFF},
			expected: 32767,
		},
		{
			name:     "2バイト変換 (-32768)",
			input:    []byte{0x80, 0x00},
			expected: -32768,
		},
		{
			name:     "3バイト変換 (8388607)",
			input:    []byte{0x7F, 0xFF, 0xFF},
			expected: 8388607,
		},
		{
			name:     "3バイト変換 (-8388608)",
			input:    []byte{0x80, 0x00, 0x00},
			expected: -8388608,
		},
		{
			name:     "4バイト変換 (2147483647)",
			input:    []byte{0x7F, 0xFF, 0xFF, 0xFF},
			expected: 2147483647,
		},
		{
			name:     "4バイト変換 (-2147483648)",
			input:    []byte{0x80, 0x00, 0x00, 0x00},
			expected: -2147483648,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BytesToInt32(tt.input)
			if result != tt.expected {
				t.Errorf("BytesToInt32(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}

	// パニックテスト
	t.Run("空のスライスの場合はパニック", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("BytesToInt32([]byte{}) did not panic")
			}
		}()
		BytesToInt32([]byte{})
	})

	t.Run("5バイト以上のスライスの場合はパニック", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("BytesToInt32([]byte{0,0,0,0,0}) did not panic")
			}
		}()
		BytesToInt32([]byte{0, 0, 0, 0, 0})
	})
}

func TestFlattenBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]byte
		expected []byte
	}{
		{
			name:     "空のスライス",
			input:    [][]byte{},
			expected: []byte{},
		},
		{
			name:     "1つの空のスライス",
			input:    [][]byte{{}},
			expected: []byte{},
		},
		{
			name:     "複数の空のスライス",
			input:    [][]byte{{}, {}, {}},
			expected: []byte{},
		},
		{
			name:     "1つのスライス",
			input:    [][]byte{{1, 2, 3}},
			expected: []byte{1, 2, 3},
		},
		{
			name:     "複数のスライス",
			input:    [][]byte{{1, 2}, {3, 4}, {5, 6}},
			expected: []byte{1, 2, 3, 4, 5, 6},
		},
		{
			name:     "異なる長さのスライス",
			input:    [][]byte{{1}, {2, 3, 4}, {5, 6}},
			expected: []byte{1, 2, 3, 4, 5, 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FlattenBytes(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("FlattenBytes(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
