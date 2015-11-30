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
package protocol

import (
	"fmt"

	"github.com/bemasher/rtlamr/crc"
	"github.com/bemasher/rtldavis/dsp"
)

func NewPacketConfig(symbolLength int) (cfg dsp.PacketConfig) {
	return dsp.NewPacketConfig(
		902355835,
		19200,
		14,
		16,
		80,
		"1100101110001001",
	)
}

type Parser struct {
	dsp.Demodulator
	crc.CRC
}

func NewParser(symbolLength int) (p Parser) {
	p.Demodulator = dsp.NewDemodulator(NewPacketConfig(symbolLength))
	p.CRC = crc.NewCRC("CCITT-16", 0, 0x1021, 0)
	return
}

func (p Parser) Cfg() dsp.PacketConfig {
	return p.Demodulator.Cfg
}

func (p Parser) Parse(pkts [][]byte) (msgs []Message) {
	seen := make(map[string]bool)

	for _, pkt := range pkts {
		for idx, b := range pkt {
			pkt[idx] = SwapBitOrder(b)
		}

		s := string(pkt)
		if seen[s] {
			continue
		}
		seen[s] = true

		// If the checksum fails, bail.
		if p.Checksum(pkt[2:]) != 0 {
			continue
		}

		msgs = append(msgs, NewMessage(pkt))
	}

	return
}

type Message struct {
	Data []byte

	ID     byte
	Sensor Sensor

	WindSpeed     byte
	WindDirection byte
}

func NewMessage(data []byte) (m Message) {
	m.Data = make([]byte, len(data)-2)
	copy(m.Data, data[2:])

	m.ID = m.Data[0] & 0xF
	m.Sensor = Sensor(m.Data[0] >> 4)
	m.WindSpeed = m.Data[1]
	m.WindDirection = m.Data[2]
	return m
}

func (m Message) String() string {
	return fmt.Sprintf("{ID:%d Sensor:%s WindSpeed:%d WindDir:%d}", m.ID, m.Sensor, m.WindSpeed, m.WindDirection)
}

type Sensor byte

const (
	UVIndex        Sensor = 4
	SolarRadiation Sensor = 6
	Light          Sensor = 7
	Temperature    Sensor = 8
	Humidity       Sensor = 0x0A
	Rain           Sensor = 0x0E
)

func (s Sensor) String() string {
	switch s {
	case UVIndex:
		return "UV Index"
	case SolarRadiation:
		return "Solar Radiation"
	case Light:
		return "Light"
	case Temperature:
		return "Temperature"
	case Humidity:
		return "Humidity"
	case Rain:
		return "Rain"
	default:
		return fmt.Sprintf("Unknown(0x%0X)", byte(s))
	}
}

func SwapBitOrder(b byte) byte {
	b = ((b & 0xF0) >> 4) | ((b & 0x0F) << 4)
	b = ((b & 0xCC) >> 2) | ((b & 0x33) << 2)
	b = ((b & 0xAA) >> 1) | ((b & 0x55) << 1)
	return b
}
