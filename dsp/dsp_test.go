package dsp

import (
	"math/rand"
	"time"

	"testing"
)

func TestRotateFs4(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	input := make([]complex128, 512)
	output := make([]complex128, 512)

	for idx := range input {
		input[idx] = complex(rand.Float64(), rand.Float64())
	}

	RotateFs4(input, output)
	RotateFs4(output, output)
	RotateFs4(output, output)
	RotateFs4(output, output)

	for idx := range input {
		if input[idx] != output[idx] {
			t.Fatalf("Failed on: %+0.6f != %+0.6f\n", input[idx], output[idx])
		}
	}
}

func BenchmarkDemodulator(b *testing.B) {
	cfg := NewPacketConfig(
		19200,
		14,
		16,
		79,
		"1100101110001001",
	)
	d := NewDemodulator(&cfg)

	block := make([]byte, d.Cfg.BlockSize2)

	b.SetBytes(int64(d.Cfg.BlockSize))
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		d.Demodulate(block)
	}
}
