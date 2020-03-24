package frskyosd

import (
	"strings"
)

const (
	darwinPortPrefix = "/dev/cu."
)

var (
	darwinPortSkip = []string{"AirPod", "iPhone", "iPad"}
)

func portName(port string) string {
	return darwinPortPrefix + port
}

func filterPorts(ports []string) []string {
	var filtered []string
	for _, v := range ports {
		skip := false
		for _, s := range darwinPortSkip {
			if strings.Contains(v, s) {
				skip = true
				break
			}
		}
		if !skip && strings.HasPrefix(v, darwinPortPrefix) {
			filtered = append(filtered, v[len(darwinPortPrefix):])
		}
	}
	return filtered
}
