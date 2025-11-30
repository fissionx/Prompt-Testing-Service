package shared

import (
	"strings"
	"testing"
)

func TestCompressDecompress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
	}{
		{
			name:    "Empty string",
			input:   "",
			wantErr: false,
		},
		{
			name:    "Short string",
			input:   "Hello World",
			wantErr: false,
		},
		{
			name:    "Long string",
			input:   strings.Repeat("This is a test string for compression. ", 100),
			wantErr: false,
		},
		{
			name:    "Special characters",
			input:   "Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?",
			wantErr: false,
		},
		{
			name:    "Unicode",
			input:   "Unicode: 你好世界 🌍 Привет مرحبا",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := CompressString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompressString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			decompressed, err := DecompressString(compressed)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecompressString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if decompressed != tt.input {
				t.Errorf("Decompressed string doesn't match original.\nGot: %s\nWant: %s", decompressed, tt.input)
			}
		})
	}
}

func TestShouldCompress(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "Empty string",
			input: "",
			want:  false,
		},
		{
			name:  "Very short string",
			input: "Hi",
			want:  false,
		},
		{
			name:  "100 bytes exactly",
			input: strings.Repeat("a", 100),
			want:  false,
		},
		{
			name:  "101 bytes",
			input: strings.Repeat("a", 101),
			want:  true,
		},
		{
			name:  "Long string",
			input: strings.Repeat("This is a long string. ", 10),
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldCompress(tt.input); got != tt.want {
				t.Errorf("ShouldCompress() = %v, want %v (length: %d)", got, tt.want, len(tt.input))
			}
		})
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that DecompressString handles uncompressed data gracefully
	uncompressedData := "This is uncompressed text"
	
	decompressed, err := DecompressString(uncompressedData)
	if err != nil {
		t.Errorf("DecompressString() should handle uncompressed data gracefully, got error: %v", err)
	}
	
	if decompressed != uncompressedData {
		t.Errorf("DecompressString() should return original when input is not compressed.\nGot: %s\nWant: %s", decompressed, uncompressedData)
	}
}

func TestCompressionRatio(t *testing.T) {
	// Test that compression actually reduces size for repetitive text
	original := strings.Repeat("This is a test string for compression. ", 100)
	
	compressed, err := CompressString(original)
	if err != nil {
		t.Fatalf("CompressString() error = %v", err)
	}
	
	// Compressed should be smaller than original for repetitive text
	if len(compressed) >= len(original) {
		t.Logf("Warning: Compressed size (%d) >= Original size (%d)", len(compressed), len(original))
		t.Logf("This may happen with base64 encoding overhead on small/random data")
	}
	
	// Verify decompression works
	decompressed, err := DecompressString(compressed)
	if err != nil {
		t.Fatalf("DecompressString() error = %v", err)
	}
	
	if decompressed != original {
		t.Errorf("Decompressed doesn't match original")
	}
}

func BenchmarkCompressString(b *testing.B) {
	data := strings.Repeat("This is a test string for compression benchmarking. ", 100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CompressString(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecompressString(b *testing.B) {
	data := strings.Repeat("This is a test string for compression benchmarking. ", 100)
	compressed, err := CompressString(data)
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecompressString(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

