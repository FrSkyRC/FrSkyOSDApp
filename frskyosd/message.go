package frskyosd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// TVStandard indicates the type of the analog TV signal
type TVStandard uint8

const (
	// TVStandardNTSC indicates that the TV signal is set to NTSC
	TVStandardNTSC = iota + 1
	// TVStandardPAL indicates that the TV signal is set to PAL
	TVStandardPAL
)

type message interface {
	frameType() frameType
	decode(cmd int, payload []byte) error
	command() int
}

// RawMessage is an undecoded message with its command
// and payload.
type RawMessage struct {
	Cmd     int
	Payload []byte
}

func (m *RawMessage) frameType() frameType { return frameTypeOSD }
func (m *RawMessage) decode(cmd int, payload []byte) error {
	m.Cmd = cmd
	m.Payload = make([]byte, len(payload))
	copy(m.Payload, payload)
	return nil
}

func (m *RawMessage) command() int { return m.Cmd }

// InfoMessage is returned in response to the INFO command
// and contains the current OSD runtime state and configuration
type InfoMessage struct {
	Version struct {
		Major uint8
		Minor uint8
		Patch uint8
	}
	Grid struct {
		Rows    uint8
		Columns uint8
	}
	Pixels struct {
		Width  uint16
		Height uint16
	}
	TVStandard        TVStandard
	HasDetectedCamera bool
	MaxFrameSize      uint16
	ContextStackSize  uint8
	IsBootloader      bool
}

func (m *InfoMessage) frameType() frameType { return frameTypeOSD }
func (m *InfoMessage) decode(cmd int, payload []byte) error {
	if len(payload) == 1 && payload[0] == 'B' {
		m.IsBootloader = true
		return nil
	}
	if len(payload) < 3 || payload[0] != 'A' || payload[1] != 'G' || payload[2] != 'H' {
		return errors.New("invalid magic number")
	}
	// Add a extra zero for the IsBootloader field
	infoPayload := append(payload[3:], 0)
	r := bytes.NewReader(infoPayload)
	return binary.Read(r, binary.LittleEndian, m)
}

func (m *InfoMessage) command() int { return int(cmdInfo) }

// FontCharMessage is returned in response to a FONT_READ request.
// It contains the character address that was requested, its data
// and its metadata.
type FontCharMessage struct {
	Addr     uint16
	Data     [54]byte
	Metadata [10]byte
}

func (m *FontCharMessage) frameType() frameType { return frameTypeOSD }
func (m *FontCharMessage) decode(cmd int, payload []byte) error {
	const (
		expectedSize = 66
	)
	if len(payload) != expectedSize {
		return fmt.Errorf("invalid payload size %d, expecing %d", len(payload), expectedSize)
	}
	return binary.Read(bytes.NewReader(payload), binary.LittleEndian, m)
}

func (m *FontCharMessage) command() int { return int(cmdReadFont) }

// ErrorMessage can be returned by methods that could fail.
// It contains the command for the request that originated it
// as well as an error code.
type ErrorMessage struct {
	Cmd   int
	Error int
}

func (m *ErrorMessage) frameType() frameType { return frameTypeOSD }
func (m *ErrorMessage) decode(cmd int, payload []byte) error {
	const (
		expectedSize = 2
	)
	if len(payload) != expectedSize {
		return fmt.Errorf("invalid payload size %d, expecing %d", len(payload), expectedSize)
	}
	m.Cmd = int(payload[0])
	m.Error = int(int8(payload[1]))
	return nil
}

func (m *ErrorMessage) command() int { return 0 }

func getMessage(t frameType, cmd int) message {
	if t == frameTypeOSD {
		switch osdCmd(cmd) {
		case cmdError:
			return &ErrorMessage{}
		case cmdInfo:
			return &InfoMessage{}
		case cmdReadFont:
			return &FontCharMessage{}
		}
		return &RawMessage{}
	}
	if t == frameTypeMSP {
		switch mspCmd(cmd) {
		case mspCmdFCVariant:
			return &mspFCVariantMessage{}
		case mspCmdFCVersion:
			return &mspFCVersionMessage{}
		case mspCmdLog:
			return &mspLogMessage{}
		}
		return &mspRawMessage{}
	}
	return nil
}
