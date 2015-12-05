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
package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bemasher/rtldavis/protocol"
	"github.com/jpoirier/gortlsdr"
)

type HopPattern []map[int]int

func NewHopPattern(n int) HopPattern {
	h := make(HopPattern, n)
	for idx := range h {
		h[idx] = make(map[int]int)
	}
	return h
}

func (h HopPattern) String() string {
	var elements []string
	for _, hop := range h {
		var hopElements []string
		for channel, count := range hop {
			hopElements = append(hopElements, fmt.Sprintf("%v:%v", channel, count))
		}
		elements = append(elements, "["+strings.Join(hopElements, ",")+"]")
	}

	return "[" + strings.Join(elements, " ") + "]"
}

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
	rand.Seed(time.Now().UnixNano())
}

func main() {
	p := protocol.NewParser(14, 0)
	p.Cfg().Log()

	fs := p.Cfg().SampleRate

	dev, err := rtlsdr.Open(0)
	if err != nil {
		log.Fatal(err)
	}

	ch := p.RandChannel()
	if err := dev.SetCenterFreq(ch); err != nil {
		log.Fatal(err)
	}

	if err := dev.SetSampleRate(fs); err != nil {
		log.Fatal(err)
	}

	if err := dev.SetTunerGainMode(false); err != nil {
		log.Fatal(err)
	}

	if err := dev.ResetBuffer(); err != nil {
		log.Fatal(err)
	}

	in, out := io.Pipe()

	go dev.ReadAsync(func(buf []byte) {
		out.Write(buf)
	}, nil, 1, p.Cfg().BlockSize2)

	// Handle frequency hops concurrently since the callback will stall if
	// we stop reading to hop.
	nextChannel := make(chan int)
	go func() {
		for ch := range nextChannel {
			if err := dev.SetCenterFreq(ch); err != nil {
				log.Fatal(err)
			}
		}
	}()

	defer func() {
		in.Close()
		out.Close()
		dev.CancelAsync()
		dev.Close()
		os.Exit(0)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	block := make([]byte, p.Cfg().BlockSize2)
	var (
		dwellTimer <-chan time.Time
		missed     bool
		missCount  int
	)

	for {
		select {
		case <-sig:
			return
		case <-dwellTimer:
			if missed {
				dwellTimer = time.After(p.DwellTime)
			}
			missed = true
			missCount++
			if missCount >= 3 {
				// We've missed three channels in a row, park on a random one
				// until we receive another message before we start hopping
				// again.
				nextChannel <- p.RandChannel()
				dwellTimer = nil
			} else {
				nextChannel <- p.NextChannel()
			}
		default:
			in.Read(block)

			recvPacket := false
			for _, msg := range p.Parse(p.Demodulate(block)) {
				recvPacket = true
				log.Printf("%02X\n", msg.Data)
			}

			if recvPacket {
				missed = false
				missCount = 0
				dwellTimer = time.After(p.DwellTime + p.DwellTime>>1)
				nextChannel <- p.NextChannel()
			}
		}
	}
}
