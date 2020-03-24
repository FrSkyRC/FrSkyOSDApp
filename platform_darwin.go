package main

// #cgo CFLAGS: -x objective-c -mmacosx-version-min=10.10
// #cgo LDFLAGS: -framework Cocoa -mmacosx-version-min=10.10
// void platform_darwin_init(void);
// void platform_darwin_after_file_dialog(void);
import "C"

func platformInit() {
	C.platform_darwin_init()
}

func platformAfterFileDialog() {
	C.platform_darwin_after_file_dialog()
}
