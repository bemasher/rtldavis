package dft

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"testing"
	"time"

	mrand "math/rand"

	"github.com/bemasher/rtldavis/dsp"
)

const (
	BlockSize = 512
	MaxErr    = 1e-12
)

func init() {
	mrand.Seed(time.Now().UnixNano())
}

func TestData(t *testing.T) {
	signalFile, err := os.Open("samples.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer signalFile.Close()

	signalBuf := bufio.NewReader(signalFile)

	outputFile, err := os.Create("output.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer outputFile.Close()

	block := make([]byte, BlockSize<<1)
	lut := dsp.NewByteToCmplxLUT()
	cmplxSignal := make([]complex128, BlockSize+N)
	sdft := NewSDFT(N, BlockSize)
	demod := make([]float64, BlockSize)

	for {
		_, err := signalBuf.Read(block)
		bailAfterEOF := err == io.EOF

		if err != nil && err != io.EOF {
			t.Fatal(err)
		}

		lut.Execute(block, cmplxSignal[N:])
		// sdft.ExecuteUnroll(cmplxSignal)
		sdft.Demod(cmplxSignal, demod)
		copy(cmplxSignal, cmplxSignal[BlockSize:])

		err = binary.Write(outputFile, binary.LittleEndian, demod)
		if err != nil {
			t.Fatal(err)
		}

		// for _, out := range sdft.out {
		// 	err = binary.Write(outputFile, binary.LittleEndian, cmplx.Abs(out[10])/20.0)
		// 	if err != nil {
		// 		t.Fatal(err)
		// 	}
		// }

		if bailAfterEOF {
			return
		}
	}
}

func TestUnroll(t *testing.T) {
	unroll := NewSDFT(N, N<<2)
	dft := NewSDFT(N, N<<2)
	exec := NewSDFT(N, N<<2)

	samples := make([]complex128, N*2)
	for idx := range samples {
		samples[idx] = complex(mrand.Float64(), mrand.Float64())
	}

	exec.Execute(samples)
	dft.ExecuteNaive(samples)
	unroll.ExecuteUnroll(samples)

	for idx := range exec.out[:32] {
		t.Logf("%+0.6f\n", exec.out[idx][:4])
		t.Logf("%+0.6f\n", dft.out[idx][:4])
		t.Logf("%+0.6f\n\n", unroll.out[idx][:4])
	}
}

func BenchmarkDFTN(b *testing.B) {
	input := make([]complex128, BlockSize+N)
	output := make([]complex128, BlockSize)

	for idx := range input {
		input[idx] = complex(mrand.Float64(), mrand.Float64())
	}

	b.SetBytes(BlockSize)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for i := 0; i < BlockSize; i++ {
			DFT14(input[i:i+N], output)
		}
	}
}

func BenchmarkSDFT(b *testing.B) {
	sdft := NewSDFT(N, BlockSize)

	input := make([]complex128, BlockSize+N)

	for idx := range input {
		input[idx] = complex(mrand.Float64(), mrand.Float64())
	}

	b.SetBytes(BlockSize)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		sdft.Execute(input)
	}
}

func BenchmarkSDFTUnroll(b *testing.B) {
	sdft := NewSDFT(N, BlockSize)

	input := make([]complex128, BlockSize+N)

	for idx := range input {
		input[idx] = complex(mrand.Float64(), mrand.Float64())
	}

	b.SetBytes(BlockSize)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		sdft.ExecuteUnroll(input)
	}
}

func BenchmarkDemod(b *testing.B) {
	sdft := NewSDFT(N, BlockSize)

	input := make([]complex128, BlockSize+N)
	output := make([]float64, BlockSize)

	for idx := range input {
		input[idx] = complex(mrand.Float64(), mrand.Float64())
	}

	b.SetBytes(BlockSize)
	b.ReportAllocs()
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		sdft.Demod(input, output)
	}
}
