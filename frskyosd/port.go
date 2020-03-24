// +build !darwin

package frskyosd

func portName(port string) string {
	return port
}

func filterPorts(ports []string) []string {
	return ports
}
