package frskyosd

import "fmt"

type frameType int

const (
	frameTypeOSD frameType = iota + 1
	frameTypeMSP
)

func (f frameType) String() string {
	switch f {
	case frameTypeOSD:
		return "OSD"
	case frameTypeMSP:
		return "MSP"
	}
	return fmt.Sprintf("unknown frameType %d", int(f))
}

type frame struct {
	Type    frameType
	Cmd     int
	Payload []byte
}
