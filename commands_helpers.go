package main

import (
	"fmt"
)

// join bulk strings words in an array of type []any into a single []byte array
func joinWords(strings []any, sep string) ([]byte, error) {
	ans := []byte{}
	for i, str := range strings {
		bStr, ok := str.(bulkString)
		if !ok {
			return nil, fmt.Errorf("%q is not of type bulkString", str)
		}
		ans = append(ans, bStr.word...)
		if sep != "" && i + 1 < len(strings) - 1 {
			ans = append(ans, []byte(sep)...)
		}
	}

	return encodeBulkString(ans), nil
}