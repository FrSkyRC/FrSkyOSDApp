GO_SRC		:= $(wildcard *.go)
ICON_SRC	:= Icon.png
ALL_SRC		:= $(GO_SRC) $(ICON_SRC)
APP_NAME 	:= FrSky OSD
BINARY_NAME	:= FrSkyOSD
APP_ID		:= com.frsky-rc.OSD
APP_VERSION := $(shell grep appVersion main.go |head -n 1 |cut -d \" -f 2)

underscore	:= _
empty		:=
space		:= $(empty) $(empty)

DIST				:= dist
DIST_NAME 			:= $(subst $(space),$(underscore),$(APP_NAME))
MACOS_DIST 			:= $(DIST)/$(DIST_NAME)-macOS-$(APP_VERSION).zip
LINUX_386_DIST		:= $(DIST)/$(DIST_NAME)-linux-386-$(APP_VERSION).tar.gz
LINUX_AMD64_DIST	:= $(DIST)/$(DIST_NAME)-linux-amd64-$(APP_VERSION).tar.gz
LINUX_ARM_DIST		:= $(DIST)/$(DIST_NAME)-linux-arm-$(APP_VERSION).tar.gz
LINUX_ARM64_DIST	:= $(DIST)/$(DIST_NAME)-linux-arm64-$(APP_VERSION).tar.gz
# This needs some work on fyne-cross
#LINUX_ALL_DIST		:= $(LINUX_386_DIST) $(LINUX_AMD64_DIST) $(LINUX_ARM_DIST) $(LINUX_ARM64_DIST)
LINUX_ALL_DIST		:= $(LINUX_AMD64_DIST)
WIN32_DIST			:= $(DIST)/$(DIST_NAME)-win32-$(APP_VERSION).zip
WIN64_DIST			:= $(DIST)/$(DIST_NAME)-win64-$(APP_VERSION).zip
WIN_ALL_DIST		:= $(WIN32_DIST) $(WIN64_DIST)

FYNE_CROSS_DIST		:= fyne-cross/dist

.PHONY: all clean

all: $(MACOS_DIST) $(LINUX_ALL_DIST) $(WIN_ALL_DIST)

$(MACOS_DIST): $(ALL_SRC)
	fyne package -appID $(APP_ID) -icon $(ICON_SRC) -name "$(APP_NAME)" -release
	macapptool sign "$(APP_NAME).app"
	ditto -c -k --sequesterRsrc --keepParent "$(APP_NAME).app" $@
	rm -r "$(APP_NAME).app"

$(DIST_NAME)-linux-%: $(ALL_SRC)
	$(eval arch := $(shell echo $@ | cut -d '-' -f 3))
	fyne-cross -docker-image docker/Dockerfile -icon $(ICON_SRC) -output "$(BINARY_NAME)" --targets=linux/$(arch)
	mkdir -p $(dir $@)
	mv "$(FYNE_CROSS_DIST)/linux-$(arch)/$(BINARY_NAME).tar.gz" "$@"

$(WIN32_DIST): $(ALL_SRC)
	fyne-cross -icon $(ICON_SRC) -output "$(BINARY_NAME)" --targets=windows/386
	mv $(FYNE_CROSS_DIST)/windows-386/$(BINARY_NAME).exe "$(APP_NAME).exe"
	mkdir -p $(dir $@)
	zip -9 "$@" "$(APP_NAME).exe"
	$(RM) "$(APP_NAME).exe"

$(WIN64_DIST): $(ALL_SRC)
	fyne-cross -icon $(ICON_SRC) -output "$(BINARY_NAME)" --targets=windows/amd64
	mv $(FYNE_CROSS_DIST)/windows-amd64/$(BINARY_NAME).exe "$(APP_NAME).exe"
	mkdir -p $(dir $@)
	zip -9 "$@" "$(APP_NAME).exe"
	$(RM) "$(APP_NAME).exe"

clean:
	$(RM) -r $(DIST)
