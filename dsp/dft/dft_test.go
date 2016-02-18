package dft

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	mrand "math/rand"
)

const (
	BlockSize = 512
	MaxErr    = 1e-12
)

func init() {
	mrand.Seed(time.Now().UnixNano())
}

type byteToCmplxLUT [256]float64

func newByteToCmplxLUT() (lut byteToCmplxLUT) {
	for idx := range lut {
		lut[idx] = (float64(idx) - 127.4) / 127.6
	}
	return lut
}

func (l *byteToCmplxLUT) Execute(in []byte, out []complex128) {
	if len(in) != len(out)<<1 {
		panic(fmt.Errorf("Incompatible slice lengths: %d, %d", len(in), len(out)))
	}

	for idx := range out {
		inIdx := idx << 1
		out[idx] = complex(l[in[inIdx]], l[in[inIdx+1]])
	}
}

func TestData(t *testing.T) {
	signalFile, err := os.Open(`laramie.bin`)
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
	lut := newByteToCmplxLUT()
	cmplxSignal := make([]complex128, BlockSize+N)
	var sdft SDFT
	demod := make([]float64, BlockSize)

	for {
		_, err := signalBuf.Read(block)
		bailAfterEOF := err == io.EOF

		if err != nil && err != io.EOF {
			t.Fatal(err)
		}

		lut.Execute(block, cmplxSignal[N:])
		sdft.Demod(cmplxSignal, demod)
		copy(cmplxSignal, cmplxSignal[BlockSize:])

		err = binary.Write(outputFile, binary.LittleEndian, demod)
		if err != nil {
			t.Fatal(err)
		}

		if bailAfterEOF {
			return
		}
	}
}

func TestSDFT5(t *testing.T) {
	input := []complex128{0, 0, 0, 0, 0, 1, 2, 3, 4, 5}
	output := make([][]complex128, 5+1)

	for idx := range output {
		output[idx] = make([]complex128, 5)
	}

	sdft := newSlidingDft(5)
	sdft.Execute(input, output)
	t.Logf("Input: %+0.0f\n", input)
	for idx, out := range output {
		t.Logf("%2d: %+0.3f\n", idx, out)

	}
}

func TestSDFT(t *testing.T) {
	input := make([]complex128, 512+14)

	dftOutput := make([][]complex128, 512+1)
	sdftOutput := make([][]complex128, 512+1)
	sdftUnrollOutput := make([][]complex128, 512+1)
	for idx := range sdftOutput {
		dftOutput[idx] = make([]complex128, 14)
		sdftOutput[idx] = make([]complex128, 14)
		sdftUnrollOutput[idx] = make([]complex128, 14)
	}

	for idx := range input {
		input[idx] = complex(mrand.Float64(), mrand.Float64())
	}

	sdft := newSlidingDft(14)
	sdft.Execute(input, sdftOutput)

	sdft = newSlidingDft(14)
	sdft.ExecuteUnroll(input, sdftUnrollOutput)

	naiveSDFT14(input, dftOutput)

	for idx := range sdftOutput[:16] {
		t.Logf("%+0.3f\n", dftOutput[idx][:4])
		t.Logf("%+0.3f\n", sdftOutput[idx][:4])
		t.Logf("%+0.3f\n\n", sdftUnrollOutput[idx][:4])
	}
}

func BenchmarkDemod(b *testing.B) {
	var sdft SDFT

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
