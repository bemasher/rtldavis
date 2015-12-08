/*
   rtldavis, an rtl-sdr receiver for Davis Instruments weather stations.
   Copyright (C) 2015  Douglas Hall

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/
package dsp

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"math/cmplx"
)

type ByteToCmplxLUT [256]float64

func NewByteToCmplxLUT() (lut ByteToCmplxLUT) {
	for idx := range lut {
		lut[idx] = (float64(idx) - 127.4) / 127.6
	}
	return lut
}

func (l *ByteToCmplxLUT) Execute(in []byte, out []complex128) {
	if len(in) != len(out)<<1 {
		panic(fmt.Errorf("Incompatible slice lengths: %d, %d", len(in), len(out)))
	}

	for idx := range out {
		inIdx := idx << 1
		out[idx] = complex(l[in[inIdx]], l[in[inIdx+1]])
	}
}

func RotateFs4(in, out []complex128) {
	for idx := 0; idx < len(out); idx += 4 {
		inAt := in[idx:]
		i0 := inAt[0]
		i1 := inAt[1]
		i2 := inAt[2]
		i3 := inAt[3]

		o1 := complex(-imag(i1), real(i1))
		o3 := complex(imag(i3), -real(i3))

		outAt := out[idx:]
		outAt[0] = i0
		outAt[1] = o1
		outAt[2] = -i2
		outAt[3] = o3
	}
}

func FIR9(in, out []complex128) {
	const (
		c0 = 0.017682261285
		c1 = 0.048171339939
		c2 = 0.122424706672
		c3 = 0.197408519126
		c4 = 0.228626345955
	)

	for idx := 0; idx < len(in)-9; idx++ {
		window := in[idx:]
		acc := (window[0] + window[8]) * c0
		acc += (window[1] + window[7]) * c1
		acc += (window[2] + window[6]) * c2
		acc += (window[3] + window[5]) * c3
		acc += window[4] * c4
		out[idx] = acc
	}
}

func Discriminate(in []complex128, out []float64) {
	// We spend a lot of time in this function and for the sake of efficiency, this:
	//     out[idx] = cmplx.Phase(in[idx] * cmplx.Conj(in[idx+1]))
	// Is equivalent to this:
	//     out[idx] = imag(in[idx] * cmplx.Conj(in[idx+1])) / cmplx.Abs(in[idx])
	// Because the magnitude of our signal should be constant, we can do this:
	//     out[idx] = imag(in[idx] * cmplx.Conj(in[idx+1]))
	// Which, if you do all the simplification, should be two multiplies and
	// an add. Need to benchmark on an RPi or RPi2 but on my desktop this is
	// almost a full order of magnitude faster.
	for idx := range out {
		out[idx] = imag(in[idx] * cmplx.Conj(in[idx+1]))
	}
}

func Quantize(input []float64, output []byte) {
	for idx, val := range input {
		output[idx] = byte(math.Float64bits(val) >> 63)
	}

	return
}

func (d *Demodulator) Pack(input []byte, slices [][]byte) {
	for symbolOffset, slice := range slices {
		for symbolIdx := range slice {
			slice[symbolIdx] = input[symbolIdx*d.Cfg.SymbolLength+symbolOffset]
		}
	}

	return
}

func (d *Demodulator) Search() (indexes []int) {
	preambleLength := len(d.Cfg.Preamble)
	for symbolOffset, slice := range d.slices {
		for symbolIdx := range slice[:len(slice)-preambleLength] {
			if bytes.Equal(d.Cfg.PreambleBytes, slice[symbolIdx:][:preambleLength]) {
				indexes = append(indexes, symbolIdx*d.Cfg.SymbolLength+symbolOffset)
			}
		}
	}

	return
}

func (d *Demodulator) Slice(indices []int) (pkts [][]byte) {
	// We will likely find multiple instances of the message so only keep
	// track of unique instances.
	seen := make(map[string]bool)

	// For each of the indices the preamble exists at.
	for _, qIdx := range indices {
		// Check that we're still within the first sample block. We'll catch
		// the message on the next sample block otherwise.
		if qIdx > d.Cfg.BlockSize {
			continue
		}

		// Packet is 1 bit per byte, pack to 8-bits per byte.
		for pIdx := 0; pIdx < d.Cfg.PacketSymbols; pIdx++ {
			d.pkt[pIdx>>3] <<= 1
			d.pkt[pIdx>>3] |= d.Quantized[qIdx+(pIdx*d.Cfg.SymbolLength)]
		}

		// Store the packet in the seen map and append to the packet list.
		pktStr := fmt.Sprintf("%02X", d.pkt)
		if !seen[pktStr] {
			seen[pktStr] = true
			pkts = append(pkts, make([]byte, len(d.pkt)))
			copy(pkts[len(pkts)-1], d.pkt)
		}
	}

	return
}

// PacketConfig specifies packet-specific radio configuration.
type PacketConfig struct {
	BitRate                        int
	SymbolLength                   int
	PreambleSymbols, PacketSymbols int

	Preamble      string
	PreambleBytes []byte

	SampleRate                   int
	BlockSize, BlockSize2        int
	PreambleLength, PacketLength int
	BufferLength                 int
}

func NewPacketConfig(bitRate, symbolLength, preambleSymbols, packetSymbols int, preamble string) PacketConfig {
	var cfg PacketConfig

	cfg.BitRate = bitRate
	cfg.SymbolLength = symbolLength

	cfg.PreambleSymbols = preambleSymbols
	cfg.PacketSymbols = packetSymbols

	cfg.PreambleLength = cfg.PreambleSymbols * cfg.SymbolLength
	cfg.PacketLength = cfg.PacketSymbols * cfg.SymbolLength

	// Pre-calculate a byte-slice version of the preamble for searching.
	cfg.Preamble = preamble
	cfg.PreambleBytes = make([]byte, len(cfg.Preamble))
	for idx := range cfg.Preamble {
		if cfg.Preamble[idx] == '1' {
			cfg.PreambleBytes[idx] = 1
		}
	}

	cfg.SampleRate = cfg.BitRate * cfg.SymbolLength

	cfg.BlockSize = 512
	cfg.BlockSize2 = cfg.BlockSize << 1

	cfg.BufferLength = (cfg.PacketLength/cfg.BlockSize + 1) * cfg.BlockSize
	if cfg.BufferLength == cfg.BlockSize {
		cfg.BufferLength += cfg.BlockSize
	}

	return cfg
}

func (cfg PacketConfig) Log() {
	log.Println("BitRate:", cfg.BitRate)
	log.Println("SymbolLength:", cfg.SymbolLength)
	log.Println("SampleRate:", cfg.SampleRate)
	log.Println("Preamble:", cfg.Preamble)
	log.Println("PreambleSymbols:", cfg.PreambleSymbols)
	log.Println("PreambleLength:", cfg.PreambleLength)
	log.Println("PacketSymbols:", cfg.PacketSymbols)
	log.Println("PacketLength:", cfg.PacketLength)
	log.Println("BlockSize:", cfg.BlockSize)
	log.Println("BufferLength:", cfg.BufferLength)
}

type Demodulator struct {
	Cfg PacketConfig

	Raw           []byte
	IQ            []complex128
	Filtered      []complex128
	Discriminated []float64
	Quantized     []byte

	slices [][]byte
	pkt    []byte

	lut ByteToCmplxLUT
}

func (d *Demodulator) Reset() {
	for idx := range d.Raw {
		d.Raw[idx] = 0
	}
	for idx := range d.IQ {
		d.IQ[idx] = 0
	}
	for idx := range d.Filtered {
		d.Filtered[idx] = 0
	}
	for idx := range d.Discriminated {
		d.Discriminated[idx] = 0
	}
	for idx := range d.Quantized {
		d.Quantized[idx] = 0
	}
}

func NewDemodulator(cfg PacketConfig) (d Demodulator) {
	d.Cfg = cfg

	d.Raw = make([]byte, d.Cfg.BufferLength<<1)
	d.IQ = make([]complex128, d.Cfg.BlockSize+9)
	d.Filtered = make([]complex128, d.Cfg.BlockSize+1)
	d.Discriminated = make([]float64, d.Cfg.BlockSize)
	d.Quantized = make([]byte, d.Cfg.BufferLength)

	d.slices = make([][]byte, d.Cfg.SymbolLength)
	flat := make([]byte, d.Cfg.BufferLength-(d.Cfg.BufferLength%d.Cfg.SymbolLength))

	symbolsPerBlock := d.Cfg.BlockSize / d.Cfg.SymbolLength
	for symbolOffset := range d.slices {
		lower := symbolOffset * symbolsPerBlock
		upper := (symbolOffset + 1) * symbolsPerBlock
		d.slices[symbolOffset] = flat[lower:upper]
	}

	d.pkt = make([]byte, (d.Cfg.PacketSymbols+7)>>3)

	d.lut = NewByteToCmplxLUT()

	return d
}

func (d *Demodulator) Demodulate(input []byte) [][]byte {
	copy(d.Raw, d.Raw[d.Cfg.BlockSize2:])
	// Only need the last filter-length worth of samples.
	// d.IQ is BlockSize + 9 for our case.
	copy(d.IQ, d.IQ[d.Cfg.BlockSize:])
	d.Filtered[0] = d.Filtered[len(d.Filtered)-1]
	copy(d.Quantized, d.Quantized[d.Cfg.BlockSize:])

	copy(d.Raw[d.Cfg.BufferLength<<1-d.Cfg.BlockSize2:], input)

	d.lut.Execute(d.Raw[d.Cfg.BufferLength<<1-d.Cfg.BlockSize2:], d.IQ[9:])
	RotateFs4(d.IQ[9:], d.IQ[9:])
	FIR9(d.IQ, d.Filtered[1:])
	Discriminate(d.Filtered, d.Discriminated)
	Quantize(d.Discriminated, d.Quantized[d.Cfg.BufferLength-d.Cfg.BlockSize:])
	d.Pack(d.Quantized[:d.Cfg.BlockSize], d.slices)
	return d.Slice(d.Search())
}
