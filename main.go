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
	"flag"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/bemasher/rtldavis/protocol"
	"github.com/jpoirier/gortlsdr"
)

var (
	id      *int
	verbose *bool

	verboseLogger *log.Logger
)

func init() {
	log.SetFlags(log.Lmicroseconds)
	rand.Seed(time.Now().UnixNano())

	id = flag.Int("id", 0, "id of the station to listen for")
	verbose = flag.Bool("v", false, "log extra information to /dev/stderr")

	flag.Parse()

	verboseLogger = log.New(ioutil.Discard, "", log.Lshortfile|log.Lmicroseconds)
	if *verbose {
		verboseLogger.SetOutput(os.Stderr)
	}
}

func main() {
	p := protocol.NewParser(14, *id)
	p.Cfg.Log()

	fs := p.Cfg.SampleRate

	dev, err := rtlsdr.Open(0)
	if err != nil {
		log.Fatal(err)
	}

	hop := p.RandHop()
	verboseLogger.Println(hop)
	if err := dev.SetCenterFreq(hop.ChannelFreq); err != nil {
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
	}, nil, 1, p.Cfg.BlockSize2)

	// Handle frequency hops concurrently since the callback will stall if we
	// stop reading to hop.
	nextHop := make(chan protocol.Hop, 1)
	go func() {
		for hop := range nextHop {
			verboseLogger.Printf("Hop: %s\n", hop)
			if err := dev.SetCenterFreq(hop.ChannelFreq + hop.FreqError); err != nil {
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

	outputFile, err := os.Create("captured.bin")
	if err != nil {
		log.Fatal(err)
	}
	defer outputFile.Close()

	block := make([]byte, p.Cfg.BlockSize2)

	// Set the dwellTimer for one full rotation of the pattern + 1. Some channels
	// may have enough frequency error that they won't receive until we've
	// seen at least one message and set the frequency correction.
	dwellTimer := time.After(52 * p.DwellTime)
	// We set missCount to 3 so that we immediately pick another random
	// channel and wait on that channel instead of hopping like we missed one.
	missCount := 3

	for {
		select {
		case <-sig:
			return
		case <-dwellTimer:
			// If the dwellTimer has expired one of two things has happened:
			//     1: We've missed a message.
			//     2: We've waited for sync and nothing has happened for a
			//        full cycle of the pattern.

			// Reset the timer and incrmeent the missed packet counter.
			dwellTimer = time.After(p.DwellTime)
			missCount++

			if missCount >= 3 {
				// We've missed three packets in a row, hop to a random
				// channel and wait for a full hopping cycle.
				nextHop <- p.RandHop()
				dwellTimer = time.After(52 * p.DwellTime)
			} else {
				// We've missed fewer than three packets in a row, hop to the
				// next channel in the pattern.
				nextHop <- p.NextHop()
			}
		default:
			in.Read(block)

			recvPacket := false
			for _, msg := range p.Parse(p.Demodulate(block)) {
				// if int(msg.ID) != *id {
				// 	continue
				// }

				outputFile.Write(p.Raw)

				recvPacket = true
				log.Printf("%02X %d\n", msg.Data, msg.ID)
			}

			if recvPacket {
				// Reset the missed packet counter.
				missCount = 0

				// Set the dwell timer to 1.5 * dwell time. If this timer
				// expires before we've received a packet then the missed
				// packet hopping logic will reset the timer to exactly the
				// dwell time and we then expect packets to arrive half-way
				// through the timer.
				dwellTimer = time.After(p.DwellTime + p.DwellTime>>1)

				// Hop to the next channel.
				nextHop <- p.NextHop()
			}
		}
	}
}
