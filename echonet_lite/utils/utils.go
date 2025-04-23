package utils

func Uint32ToBytes(n uint32, size int) []byte {
	if size < 1 || size > 4 {
		panic("size must be 1, 2, 3, or 4")
	}
	b := make([]byte, size)
	for i := range size {
		shift := uint((size - 1 - i) * 8)
		b[i] = byte(n >> shift)
	}
	return b
}

func BytesToUint32(b []byte) uint32 {
	size := len(b)
	if size < 1 || size > 4 {
		panic("slice length must be 1, 2, 3, or 4")
	}
	var n uint32
	for i := range size {
		shift := uint((size - 1 - i) * 8)
		n |= uint32(b[i]) << shift
	}
	return n
}

func BytesToInt32(b []byte) int32 {
	size := len(b)
	if size < 1 || size > 4 {
		panic("slice length must be 1, 2, 3, or 4")
	}
	var n int32
	if b[0]&0x80 != 0 {
		n = -1
	}
	for i := range size {
		n <<= 8
		n |= int32(b[i])
	}
	return n
}

func FlattenBytes(chunks [][]byte) []byte {
	// 合計サイズを計算
	totalSize := 0
	for _, chunk := range chunks {
		totalSize += len(chunk)
	}

	// 必要なサイズを確保
	result := make([]byte, 0, totalSize)

	// バイト列を結合
	for _, chunk := range chunks {
		result = append(result, chunk...)
	}

	return result
}
