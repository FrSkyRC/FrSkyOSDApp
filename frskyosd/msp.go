package frskyosd

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	version "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

type fc struct {
	Identifier        string
	Name              string
	MinVersion        string
	SerialFunctionBit byte
}

var (
	supportedFCs = []*fc{
		{"INAV", "INAV", "2.4.0", 20},
		{"BTFL", "Betaflight", "4.2.0", 16},
	}
)

type mspCmd int

const (
	mspCmdFCVariant      mspCmd = 2
	mspCmdFCVersion      mspCmd = 3
	mspCmdSetPassthrough mspCmd = 245
	mspCmdLog            mspCmd = 253

	mspPassthroughSerialByFunctionID = 0xfe
)

type mspRawMessage struct {
	Cmd     int
	Payload []byte
}

func (m *mspRawMessage) frameType() frameType { return frameTypeMSP }
func (m *mspRawMessage) command() int         { return m.Cmd }
func (m *mspRawMessage) decode(cmd int, payload []byte) error {
	m.Cmd = cmd
	m.Payload = payload[1:]
	return nil
}

type mspLogMessage struct {
	Message string
}

func (m *mspLogMessage) frameType() frameType { return frameTypeMSP }
func (m *mspLogMessage) command() int         { return int(mspCmdLog) }
func (m *mspLogMessage) decode(cmd int, payload []byte) error {
	// Skip direction, flag and final null byte
	m.Message = string(payload[2 : len(payload)-1])
	return nil
}

type mspFCVariantMessage struct {
	Variant string
}

func (m *mspFCVariantMessage) frameType() frameType { return frameTypeMSP }
func (m *mspFCVariantMessage) command() int         { return int(mspCmdFCVariant) }
func (m *mspFCVariantMessage) decode(cmd int, payload []byte) error {
	// Skip direction
	m.Variant = string(payload[1:])
	return nil
}

type mspFCVersionMessage struct {
	Major, Minor, Patch uint8
}

func (m *mspFCVersionMessage) frameType() frameType { return frameTypeMSP }
func (m *mspFCVersionMessage) command() int         { return int(mspCmdFCVersion) }
func (m *mspFCVersionMessage) decode(cmd int, payload []byte) error {
	m.Major = payload[1]
	m.Minor = payload[2]
	m.Patch = payload[3]
	return nil
}

func (o *OSD) sendMSP(cmd mspCmd, data []byte) error {
	log.Debugf("MSP: %d=> %s\n", cmd, hex.EncodeToString(data))

	payload := make([]byte, 0, 2+len(data))
	payload = append(payload, byte(len(data)))
	payload = append(payload, byte(cmd))
	payload = append(payload, data...)

	cs := newXorChecksum()
	checkSumWrite(cs, payload)

	var buf bytes.Buffer
	buf.WriteByte('$')
	buf.WriteByte('M')
	buf.WriteByte('<')
	buf.Write(payload)
	buf.WriteByte(cs.Sum8())

	return o.write(buf.Bytes())
}

func (o *OSD) getFCFirmware() (variant string, version string, err error) {
	if err := o.sendMSP(mspCmdFCVariant, nil); err != nil {
		return "", "", err
	}
	resp, err := o.awaitResponse()
	if err != nil {
		return "", "", err
	}
	fcVariant, ok := resp.(*mspFCVariantMessage)
	if !ok {
		return "", "", errors.New("could not retrieve FC variant")
	}
	if err := o.sendMSP(mspCmdFCVersion, nil); err != nil {
		return "", "", err
	}
	resp, err = o.awaitResponse()
	if err != nil {
		return "", "", err
	}
	fcVersion, ok := resp.(*mspFCVersionMessage)
	if !ok {
		return "", "", errors.New("could not retrieve FC version")
	}
	version = fmt.Sprintf("%d.%d.%d", fcVersion.Major, fcVersion.Minor, fcVersion.Patch)
	return fcVariant.Variant, version, nil
}

func (o *OSD) setupMspPassthrough() (bool, error) {
	fcVariant, fcVersion, err := o.getFCFirmware()
	if err != nil {
		return false, err
	}
	var targetFc *fc
	var fcNames []string
	for _, v := range supportedFCs {
		fcNames = append(fcNames, v.Name)
		if v.Identifier == fcVariant {
			targetFc = v
			break
		}
	}
	if targetFc == nil {
		return false, fmt.Errorf("can't connect via %s, use %s", fcVariant, strings.Join(fcNames, " or "))
	}
	minVer := version.Must(version.NewVersion(targetFc.MinVersion))
	fcVer := version.Must(version.NewVersion(fcVersion))
	if fcVer.LessThan(minVer) {
		return false, fmt.Errorf("can't connact via %s %s, use %s at least", targetFc.Name, fcVer, minVer)
	}
	ptPayload := []byte{mspPassthroughSerialByFunctionID, targetFc.SerialFunctionBit}
	if err := o.sendMSP(mspCmdSetPassthrough, ptPayload); err != nil {
		return false, err
	}
	resp, err := o.awaitResponse()
	if err != nil {
		return false, err
	}
	rawResp, ok := resp.(*mspRawMessage)
	if !ok || len(rawResp.Payload) == 0 {
		return false, errors.New("unknown error setting up MSP passthrough")
	}
	if rawResp.Payload[0] != 1 {
		// Port for the OSD is not configured
		return false, errors.New("no port in the FC is configured for FrSky OSD")
	}
	return true, nil
}
