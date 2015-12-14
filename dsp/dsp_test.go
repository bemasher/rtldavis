package dsp

import (
	"testing"
	"time"

	crand "crypto/rand"
	mrand "math/rand"
)

func TestRotateFs4(t *testing.T) {
	mrand.Seed(time.Now().UnixNano())

	input := make([]complex128, 512)
	output := make([]complex128, 512)

	for idx := range input {
		input[idx] = complex(mrand.Float64(), mrand.Float64())
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

func BenchmarkByteToCmplxLUT(b *testing.B) {
	lut := NewByteToCmplxLUT()

	input := make([]byte, 512)
	output := make([]complex128, 256)

	crand.Read(input)

	b.SetBytes(512)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		lut.Execute(input, output)
	}
}

func BenchmarkFIR9(b *testing.B) {
	input := make([]complex128, 512+9)
	output := make([]complex128, 512)

	for idx := range input {
		input[idx] = complex(mrand.Float64(), mrand.Float64())
	}

	b.SetBytes(512)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		FIR9(input, output)
	}
}

func BenchmarkDiscriminate(b *testing.B) {
	input := make([]complex128, 513)
	output := make([]float64, 512)

	for idx := range input {
		input[idx] = complex(mrand.Float64(), mrand.Float64())
	}

	b.SetBytes(512)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		Discriminate(input, output)
	}
}

func BenchmarkQuantize(b *testing.B) {
	input := make([]float64, 512)
	output := make([]byte, 512)

	for idx := range input {
		input[idx] = mrand.Float64()
	}

	b.SetBytes(512)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		Quantize(input, output)
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
