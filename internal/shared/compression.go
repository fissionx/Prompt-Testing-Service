package shared

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
)

// CompressString compresses a string using gzip and returns base64 encoded string
func CompressString(data string) (string, error) {
	if data == "" {
		return "", nil
	}

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	
	_, err := gzWriter.Write([]byte(data))
	if err != nil {
		return "", err
	}
	
	if err := gzWriter.Close(); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// DecompressString decompresses a base64 encoded gzipped string
func DecompressString(compressed string) (string, error) {
	if compressed == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(compressed)
	if err != nil {
		// If decode fails, assume it's uncompressed (backward compatibility)
		return compressed, nil
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		// If gzip reader fails, assume it's uncompressed (backward compatibility)
		return compressed, nil
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		// If decompression fails, return original
		return compressed, nil
	}

	return string(decompressed), nil
}

// ShouldCompress determines if data should be compressed based on size
// Only compress if larger than 100 bytes to avoid overhead
func ShouldCompress(data string) bool {
	return len(data) > 100
}

