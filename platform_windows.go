package main

import (
	"net/http"

	ieproxy "github.com/mattn/go-ieproxy"
)

func platformInit() {
	ieproxy.OverrideEnvWithStaticProxy()
	http.DefaultTransport.(*http.Transport).Proxy = http.ProxyFromEnvironment
}

func platformSetup()           {}
func platformAfterFileDialog() {}
