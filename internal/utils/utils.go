package utils

import (
	"fmt"
)

// Split a []byte into chunks of byte arrays each with size chunkSize
func SplitIntoChunks(data []byte, chunkSize int) ([][]byte, error) {
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunk size must be greater than 0")
	}

	var chunks [][]byte

	return chunks, nil
}
