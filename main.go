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
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/bemasher/rtldavis/protocol"
	"github.com/jpoirier/gortlsdr"
)

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

	if err := dev.SetCenterFreq(p.NextChannel()); err != nil {
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

	const (
		dwellTime = 2500 * time.Millisecond
	)

	var (
		last time.Time

		channelIdx int

		patternIdx int
		pattern    []map[int]int
	)

	pattern = make([]map[int]int, p.ChannelCount)
	for idx := range pattern {
		pattern[idx] = make(map[int]int)
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
					log.Println(offset)

					// Figure out where we are in the pattern relative to the last hop.
					patternIdx = (patternIdx + int(math.Floor(offset+0.5))) % p.ChannelCount
					// Increment this channel for the offset in the pattern.
					pattern[patternIdx][channelIdx]++
				}
				last = time.Now()

				// Get a new random channel and go there.
				channelIdx := rand.Intn(p.ChannelCount)
				nextChannel <- p.Channels[channelIdx]
			}
		}
	}
}
