package frskyosd

import (
	"io"
	"net"
	"os"
	"strings"

	"go.bug.st/serial"
)

const (
	tcpPrefix = "tcp:"
)

type connection interface {
	io.Reader
	io.Writer
	io.Closer
}

func openTCPConnection(addr string) (connection, error) {
	return net.Dial("tcp", addr)
}

func openSerialConnection(port string) (connection, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
	}
	return serial.Open(portName(port), mode)
}

func openConnection(name string) (connection, error) {
	if strings.HasPrefix(name, tcpPrefix) {
		return openTCPConnection(name[len(tcpPrefix):])
	}
	return openSerialConnection(name)
}

var (
	tcpPorts []string
)

// AvailablePorts returns the list of ports in the system
// that can be used to connect to FrSkyOSD
func AvailablePorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		if pe, ok := err.(*serial.PortError); ok {
			if pe.Code() == serial.ErrorEnumeratingPorts {
				// This happens on Windows when there are
				// no serial ports
				return nil, nil
			}
		}
		return nil, err
	}
	filtered := filterPorts(ports)
	filtered = append(filtered, tcpPorts...)
	return filtered, nil
}

func init() {
	if tp := os.Getenv("FRSKY_OSD_TCP_PORTS"); tp != "" {
		for _, v := range strings.Split(tp, ",") {
			tcpPorts = append(tcpPorts, tcpPrefix+v)
		}
	}
}
