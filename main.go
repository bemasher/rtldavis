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
	"math"
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
	p := protocol.NewParser(14)
	p.Cfg().Log()

	fs := p.Cfg().SampleRate

	dev, err := rtlsdr.Open(0)
	if err != nil {
		log.Fatal(err)
	}

	channelIdx := rand.Intn(p.ChannelCount)
	if err := dev.SetCenterFreq(p.Channels[channelIdx]); err != nil {
		log.Fatal(err)
	}
	log.Printf("Channel: %2d %d\n", channelIdx, p.Channels[channelIdx])

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

	const (
		dwellTime = 2562500 * time.Microsecond
	)

	var (
		last time.Time

		patternIdx int
		pattern    HopPattern
	)

	// pattern = NewHopPattern(p.ChannelCount)
	pattern = []map[int]int{
		{0: 0}, {19: 0}, {41: 0}, {25: 0}, {8: 0}, {47: 0}, {32: 0}, {13: 0}, {36: 0},
		{22: 0}, {3: 0}, {29: 0}, {44: 0}, {16: 0}, {5: 0}, {27: 0}, {38: 0}, {10: 0},
		{49: 0}, {21: 0}, {2: 0}, {30: 0}, {42: 0}, {14: 0}, {48: 0}, {7: 0}, {24: 0},
		{34: 0}, {45: 0}, {1: 0}, {17: 0}, {39: 0}, {26: 0}, {9: 0}, {31: 0}, {50: 0},
		{37: 0}, {12: 0}, {20: 0}, {33: 0}, {4: 0}, {43: 0}, {28: 0}, {15: 0},
		{35: 0}, {6: 0}, {40: 0}, {11: 0}, {23: 0}, {46: 0}, {18: 0},
	}

	for {
		select {
		case <-sig:
			return
		default:
			in.Read(block)

			recvPacket := false
			for _, msg := range p.Parse(p.Demodulate(block)) {
				recvPacket = true
				log.Printf("%02X\n", msg.Data)
			}

			if recvPacket {
				// Can't calculate hop offset if we don't have a previous message time.
				if !last.IsZero() {
					// Get time since last message.
					timeDiff := time.Since(last)
					// Get channel offset based on dwelltime per channel.
					offset := float64(timeDiff) / float64(dwellTime)
					log.Printf("%2d %0.3f\n", int(math.Floor(offset+0.5)), offset)

					// Figure out where we are in the pattern relative to the last hop.
					patternIdx = (patternIdx + int(math.Floor(offset+0.5))) % p.ChannelCount
				}
				last = time.Now()

				// Increment this channel and decrement all others in this hop.
				pattern[patternIdx][channelIdx]++
				for ch := range pattern[patternIdx] {
					if ch != channelIdx {
						pattern[patternIdx][ch]--
					}
				}

				// Prune bad channels from the pattern.
				for ch, count := range pattern[patternIdx] {
					if count < 0 {
						delete(pattern[patternIdx], ch)
					}
				}

				// If the next hop in the pattern has been previously visited,
				// hop to the most visited channel on that hop instead of a random channel.
				nextPatternIdx := (patternIdx + 1) % p.ChannelCount
				if hop := pattern[nextPatternIdx]; len(hop) != 0 {
					max := ^int(^uint(0) >> 1)
					for ch, count := range hop {
						if max < count {
							max = count
							channelIdx = ch
						}
					}
				} else {
					channelIdx = rand.Intn(p.ChannelCount)
				}
				nextChannel <- p.Channels[channelIdx]
				log.Println(pattern)
				log.Printf("Channel: %2d %d\n", channelIdx, p.Channels[channelIdx])
			}
		}
	}
}
