package main

// sliceの重複を削除したコピーを返す
func removeDuplicates[T comparable](slice []T) []T {
	encountered := map[T]bool{}
	result := make([]T, 0, len(slice))

	for v := range slice {
		if !encountered[slice[v]] {
			encountered[slice[v]] = true
			result = append(result, slice[v])
		}
	}
	return result
}
