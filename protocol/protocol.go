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
	"math"
	"math/rand"
	"time"

	"github.com/bemasher/rtlamr/crc"
	"github.com/bemasher/rtldavis/dsp"
)

func NewPacketConfig(symbolLength int) (cfg dsp.PacketConfig) {
	return dsp.NewPacketConfig(
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

	Cfg dsp.PacketConfig

	ID        int
	DwellTime time.Duration

	channelCount int
	channels     []int

	hopIdx     int
	hopPattern []int

	currentFreqErr int
	channelFreqErr map[int]int
}

func NewParser(symbolLength, id int) (p Parser) {
	p.Cfg = NewPacketConfig(symbolLength)
	p.Demodulator = dsp.NewDemodulator(&p.Cfg)
	p.CRC = crc.NewCRC("CCITT-16", 0, 0x1021, 0)

	p.channels = []int{
		902355835, 902857585, 903359336, 903861086, 904362837, 904864587,
		905366338, 905868088, 906369839, 906871589, 907373340, 907875090,
		908376841, 908878591, 909380342, 909882092, 910383843, 910885593,
		911387344, 911889094, 912390845, 912892595, 913394346, 913896096,
		914397847, 914899597, 915401347, 915903098, 916404848, 916906599,
		917408349, 917910100, 918411850, 918913601, 919415351, 919917102,
		920418852, 920920603, 921422353, 921924104, 922425854, 922927605,
		923429355, 923931106, 924432856, 924934607, 925436357, 925938108,
		926439858, 926941609, 927443359,
	}
	p.channelCount = len(p.channels)

	p.hopIdx = rand.Intn(p.channelCount)
	p.hopPattern = []int{
		0, 19, 41, 25, 8, 47, 32, 13, 36, 22, 3, 29, 44, 16, 5, 27, 38, 10,
		49, 21, 2, 30, 42, 14, 48, 7, 24, 34, 45, 1, 17, 39, 26, 9, 31, 50,
		37, 12, 20, 33, 4, 43, 28, 15, 35, 6, 40, 11, 23, 46, 18,
	}

	p.channelFreqErr = make(map[int]int)

	p.ID = id
	p.DwellTime = 2562500 * time.Microsecond
	p.DwellTime += time.Duration(p.ID) * 62500 * time.Microsecond

	return
}

type Hop struct {
	ChannelIdx  int
	ChannelFreq int
	FreqError   int
}

func (h Hop) String() string {
	return fmt.Sprintf("{ChannelIdx:%2d ChannelFreq:%d FreqError:%d}",
		h.ChannelIdx, h.ChannelFreq, h.FreqError,
	)
}

func (p *Parser) hop() (h Hop) {
	h.ChannelIdx = p.hopPattern[p.hopIdx]
	h.ChannelFreq = p.channels[h.ChannelIdx]

	// If this channel has already been visited, use frequency error from last
	// visit. Otherwise use frequency error from previous channel.
	if freqErr, exists := p.channelFreqErr[p.hopPattern[p.hopIdx]]; exists {
		p.currentFreqErr = freqErr
	}
	h.FreqError = p.currentFreqErr

	return h
}

// Increment the pattern index and return the new channel's parameters.
func (p *Parser) NextHop() Hop {
	p.hopIdx = (p.hopIdx + 1) % p.channelCount
	return p.hop()
}

// Randomize the pattern index and return the new channel's parameters.
func (p *Parser) RandHop() Hop {
	p.hopIdx = rand.Intn(p.channelCount)
	return p.hop()
}

// Given a list of packets, check them for validity and ignore duplicates,
// return a list of parsed messages.
func (p *Parser) Parse(pkts []dsp.Packet) (msgs []Message) {
	seen := make(map[string]bool)

	for _, pkt := range pkts {
		// Bit order over-the-air is reversed.
		for idx, b := range pkt.Data {
			pkt.Data[idx] = SwapBitOrder(b)
		}

		// Keep track of duplicate packets.
		s := string(pkt.Data)
		if seen[s] {
			continue
		}
		seen[s] = true

		// If the checksum fails, bail.
		if p.Checksum(pkt.Data[2:]) != 0 {
			continue
		}

		// Look at the packet's tail to determine frequency error between
		// transmitter and receiver.
		lower := pkt.Idx + 8*p.Cfg.SymbolLength
		upper := pkt.Idx + 24*p.Cfg.SymbolLength
		tail := p.Demodulator.Discriminated[lower:upper]

		var mean float64
		for _, sample := range tail {
			mean += sample
		}
		mean /= float64(len(tail))

		// The tail is a series of zero symbols. The driminator's output is
		// measured in radians.
		freqError := -int(9600 + (mean*float64(p.Cfg.SampleRate))/(2*math.Pi))

		// Set the current channel's frequency error.
		p.channelFreqErr[p.hopPattern[p.hopIdx]] = p.currentFreqErr + freqError

		// Update the current frequency error.
		p.currentFreqErr += freqError

		msgs = append(msgs, NewMessage(pkt))
	}

	return
}

type Message struct {
	dsp.Packet

	ID            byte
	BatteryStatus byte

	Sensor Sensor

	WindSpeed     byte
	WindDirection float64
}

func NewMessage(pkt dsp.Packet) (m Message) {
	m.Idx = pkt.Idx
	m.Data = make([]byte, len(pkt.Data)-2)
	copy(m.Data, pkt.Data[2:])

	m.ID = m.Data[0] & 0x7
	m.BatteryStatus = (m.Data[0] >> 3) & 1

	m.WindSpeed = m.Data[1]
	m.WindDirection = float64(m.Data[2]) / 255 * 360

	m.Sensor = NewSensor(m.Data[0]>>4, m.Data[3:5])

	return m
}

func (m Message) String() string {
	return fmt.Sprintf("{ID:%d Battery:%d WindSpeed:%2d WindDir:%3.1f %s Raw:%02X}",
		m.ID, m.BatteryStatus, m.WindSpeed, m.WindDirection, m.Sensor, m.Data,
	)
}

type Sensor struct {
	Type  byte
	Raw   [2]uint16
	Value float64
}

const (
	UVIndex        = 0x04
	SolarRadiation = 0x06
	Light          = 0x07
	Temperature    = 0x08
	Humidity       = 0x0A
	Rain           = 0x0E
)

func NewSensor(typ byte, raw []byte) (s Sensor) {
	s.Type = typ
	for idx, b := range raw {
		s.Raw[idx] = uint16(b)
	}

	switch s.Type {
	case UVIndex:
		s.Value = float64((s.Raw[0] << 4) | (s.Raw[1] >> 4))
		s.Value = (s.Value - 4) / 200.0
	case SolarRadiation:
		s.Value = float64((s.Raw[0] << 4) | (s.Raw[1] >> 4))
		s.Value = (s.Value - 4) / 2.27
	case Light:
		s.Value = float64((s.Raw[0] << 4) | (s.Raw[1] >> 4))
	case Temperature:
		s.Value = float64(int16(s.Raw[0]<<8+s.Raw[1])>>4) / 10.0
	case Humidity:
		s.Value = float64((s.Raw[1]>>4)<<8+s.Raw[0]) / 10.0
	case Rain:
		s.Value = float64(s.Raw[0])
	}

	return s
}

func (s Sensor) String() string {
	switch s.Type {
	case UVIndex:
		return fmt.Sprintf("UVIndex:%0.1f", s.Value)
	case SolarRadiation:
		return fmt.Sprintf("SolarRadiation:%0.1f", s.Value)
	case Light:
		return fmt.Sprintf("Light:%0.1f", s.Value)
	case Temperature:
		return fmt.Sprintf("Temperature:%0.1f", s.Value)
	case Humidity:
		return fmt.Sprintf("Humidity:%0.1f", s.Value)
	case Rain:
		return fmt.Sprintf("Rain:%0.0f", s.Value)
	default:
		return fmt.Sprintf("Unknown(0x%X):0x%02X", s.Type, s.Raw[:])
	}
}

func SwapBitOrder(b byte) byte {
	b = ((b & 0xF0) >> 4) | ((b & 0x0F) << 4)
	b = ((b & 0xCC) >> 2) | ((b & 0x33) << 2)
	b = ((b & 0xAA) >> 1) | ((b & 0x55) << 1)
	return b
}
