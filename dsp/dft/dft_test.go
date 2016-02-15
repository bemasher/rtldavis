package dft

import (
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

// func TestData(t *testing.T) {
// 	signalFile, err := os.Open("samples.bin")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer signalFile.Close()

// 	signalBuf := bufio.NewReader(signalFile)

// 	outputFile, err := os.Create("output.bin")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer outputFile.Close()

// 	block := make([]byte, BlockSize<<1)
// 	lut := NewByteToCmplxLUT()
// 	cmplxSignal := make([]complex128, BlockSize+N)
// 	var sdft SDFT
// 	demod := make([]float64, BlockSize)

// 	for {
// 		_, err := signalBuf.Read(block)
// 		bailAfterEOF := err == io.EOF

// 		if err != nil && err != io.EOF {
// 			t.Fatal(err)
// 		}

// 		lut.Execute(block, cmplxSignal[N:])
// 		sdft.Demod(cmplxSignal, demod)
// 		copy(cmplxSignal, cmplxSignal[BlockSize:])

// 		err = binary.Write(outputFile, binary.LittleEndian, demod)
// 		if err != nil {
// 			t.Fatal(err)
// 		}

// 		if bailAfterEOF {
// 			return
// 		}
// 	}
// }

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
