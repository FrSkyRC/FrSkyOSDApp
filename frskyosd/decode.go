package frskyosd

import (
	"bytes"
	"encoding/hex"
	"fmt"

	log "github.com/sirupsen/logrus"
)

type decoderState int

const (
	decoderStateNone decoderState = iota
	decoderStateSync

	decoderStateOSDLength
	decoderStateOSDCmd
	decoderStateOSDPayload

	decoderStateMSPv1Direction
	decoderStateMSPv1PayloadSize
	decoderStateMSPv1Command
	decoderStateMSPv1Payload

	decoderStateMSPv2Direction
	decoderStateMSPv2Flag
	decoderStateMSPv2CommandLow
	decoderStateMSPv2CommandHigh
	decoderStateMSPv2PayloadSizeLow
	decoderStateMSPv2PayloadSizeHigh
	decoderStateMSPv2Payload

	decoderStateChecksum
)

func (o *OSD) decodeResponses() {
	state := decoderStateNone
	var t frameType
	var buf bytes.Buffer
	var cmd int
	var payloadSize int
	var cs checkSum

	reset := func() {
		state = decoderStateNone
		t = 0
		buf.Reset()
		cmd = 0
		payloadSize = 0
		cs = nil
	}

	for c := range o.connCh {
		switch state {
		case decoderStateNone:
			if c != '$' {
				break
			}
			state = decoderStateSync
		case decoderStateSync:
			switch c {
			case 'A':
				state = decoderStateOSDLength
				t = frameTypeOSD
				cs = newCrc8D5Checksum()
			case 'M':
				state = decoderStateMSPv1Direction
				t = frameTypeMSP
				cs = newXorChecksum()
			case 'X':
				state = decoderStateMSPv2Direction
				t = frameTypeMSP
				cs = newCrc8D5Checksum()
			default:
				log.Warnf("unknown sync char %q", string([]byte{c}))
				reset()
			}
		case decoderStateOSDLength:
			payloadSize = int(c)
			cs.WriteByte(c)
			state = decoderStateOSDCmd
		case decoderStateOSDCmd:
			cmd = int(c)
			cs.WriteByte(c)
			if payloadSize > 0 {
				state = decoderStateOSDPayload
			} else {
				state = decoderStateChecksum
			}
		case decoderStateOSDPayload:
			buf.WriteByte(c)
			cs.WriteByte(c)
			if buf.Len() == payloadSize-1 {
				state = decoderStateChecksum
			}

		case decoderStateMSPv1Direction:
			if c != '<' && c != '>' && c != '!' {
				log.Warnf("unknown MSPv1 direction char %q", string([]byte{c}))
				reset()
				break
			}
			buf.WriteByte(c)
			state = decoderStateMSPv1PayloadSize
		case decoderStateMSPv1PayloadSize:
			cs.WriteByte(c)
			payloadSize = int(c)
			state = decoderStateMSPv1Command
		case decoderStateMSPv1Command:
			cs.WriteByte(c)
			cmd = int(c)
			if payloadSize > 0 {
				state = decoderStateMSPv1Payload
			} else {
				state = decoderStateChecksum
			}
		case decoderStateMSPv1Payload:
			cs.WriteByte(c)
			buf.WriteByte(c)
			if buf.Len() == payloadSize+1 {
				state = decoderStateChecksum
			}

		case decoderStateMSPv2Direction:
			if c != '<' && c != '>' && c != '!' {
				log.Warnf("unknown MSPv2 direction char %q", string([]byte{c}))
				reset()
				break
			}
			buf.WriteByte(c)
			state = decoderStateMSPv2Flag
		case decoderStateMSPv2Flag:
			buf.WriteByte(c)
			state = decoderStateMSPv2CommandLow
		case decoderStateMSPv2CommandLow:
			cs.WriteByte(c)
			cmd = int(c)
			state = decoderStateMSPv2CommandHigh
		case decoderStateMSPv2CommandHigh:
			cs.WriteByte(c)
			cmd |= int(c) << 8
			state = decoderStateMSPv2PayloadSizeLow
		case decoderStateMSPv2PayloadSizeLow:
			cs.WriteByte(c)
			payloadSize = int(c)
			state = decoderStateMSPv2PayloadSizeHigh
		case decoderStateMSPv2PayloadSizeHigh:
			cs.WriteByte(c)
			payloadSize |= int(c) << 8
			if payloadSize > 0 {
				state = decoderStateMSPv2Payload
			} else {
				state = decoderStateChecksum
			}
		case decoderStateMSPv2Payload:
			cs.WriteByte(c)
			buf.WriteByte(c)
			if buf.Len() == payloadSize+2 {
				state = decoderStateChecksum
			}

		case decoderStateChecksum:
			if cs.Sum8() == c {
				log.Debugf("%s:%d<= %s\n", t.String(), cmd, hex.EncodeToString(buf.Bytes()))
				o.responseCh <- &frame{
					Type:    t,
					Cmd:     cmd,
					Payload: buf.Bytes(),
				}
			} else {
				log.Warnf("invalid checksum 0x%02x vs expected 0x%02x", c, cs.Sum8())
			}
			reset()
		default:
			panic(fmt.Errorf("invalid decoder state %d", state))
		}
	}
	close(o.responseCh)
}
