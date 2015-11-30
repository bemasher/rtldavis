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
	"os"
	"os/signal"

	"github.com/bemasher/rtldavis/protocol"
	"github.com/jpoirier/gortlsdr"
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
}

func main() {
	p := protocol.NewParser(14)
	p.Cfg().Log()

	fs := p.Cfg().SampleRate
	fc := int(p.Cfg().CenterFreq)

	dev, err := rtlsdr.Open(0)
	if err != nil {
		log.Fatal(err)
	}

	if err := dev.SetCenterFreq(fc); err != nil {
		log.Fatal(err)
	}

	if err := dev.SetSampleRate(fs); err != nil {
		log.Fatal(err)
	}

	if err := dev.ResetBuffer(); err != nil {
		log.Fatal(err)
	}

	in, out := io.Pipe()

	go dev.ReadAsync(func(buf []byte) {
		out.Write(buf)
	}, nil, 1, 16384)

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

	for {
		select {
		case <-sig:
			return
		default:
			in.Read(block)

			for _, msg := range p.Parse(p.Demodulate(block)) {
				log.Printf("%02X\n", msg.Data)
			}
		}
	}
}
