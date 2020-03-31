package main // import "osdapp"

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"

	"github.com/fiam/max7456tool/mcm"
	log "github.com/sirupsen/logrus"
	dlgs "github.com/sqweek/dialog"

	"osdapp/fonts"
	"osdapp/frskyosd"
	"osdapp/internal/autoupdater"
	"osdapp/internal/dialog"
)

const (
	fontsDir   = "fonts"
	fontsExt   = ".mcm"
	appVersion = "1.0.1"

	updatesSource = "https://github.com/FrSkyRC/FrSkyOSDApp"
)

// App is an opaque type that contains the whole application state
type App struct {
	app                 fyne.App
	connected           bool
	ports               []string
	window              fyne.Window
	portsSelect         *widget.Select
	connectButton       *widget.Button
	versionLabel        *widget.Label
	uploadFontButton    *widget.Button
	fontItems           []*FontIcon
	uploadFontDialog    dialog.Dialog
	flashFirmwareButton *widget.Button
	connectedPort       string
	osd                 *frskyosd.OSD
}

func newApp() *App {
	a := &App{}
	a.app = app.New()
	a.updatePorts()
	a.connectButton = widget.NewButton("Connect", a.connectOrDisconnect)
	a.connectButton.Disable()
	a.portsSelect = widget.NewSelect(a.ports, a.portSelectionChanged)
	a.uploadFontButton = widget.NewButton("Upload Font", a.uploadFont)
	versionStyle := fyne.TextStyle{
		Monospace: true,
	}
	a.versionLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, versionStyle)
	var row []fyne.CanvasObject
	var fontRows []fyne.CanvasObject
	for ii := 0; ii < fontCharCount; ii++ {
		if ii%fontRowSize == 0 {
			if len(row) > 0 {
				hbox := widget.NewHBox(row...)
				fontRows = append(fontRows, hbox)
			}
			row = nil
		}
		fi := NewFontIcon()
		row = append(row, &fi.Image)
		a.fontItems = append(a.fontItems, fi)
	}
	a.flashFirmwareButton = widget.NewButton("Flash Firmware", a.selectFirmware)
	windowTitle := "FrSky OSD"
	if runtime.GOOS != "darwin" {
		// Neither Windows nor Linux have an obvious way
		// to see the app version. Show it in the window
		// title.
		windowTitle += fmt.Sprintf(" (%s)", appVersion)
	}
	a.window = a.app.NewWindow(windowTitle)
	a.window.SetIcon(fyne.NewStaticResource("Icon", iconBytes))
	a.window.SetContent(widget.NewVBox(
		widget.NewHBox(
			widget.NewLabel("Port:"),
			a.portsSelect,
			layout.NewSpacer(),
			a.connectButton,
		),
		widget.NewHBox(
			widget.NewLabel("Font:"),
			layout.NewSpacer(),
			a.uploadFontButton,
		),
		widget.NewVBox(fontRows...),
		layout.NewSpacer(),
		widget.NewHBox(
			widget.NewLabelWithStyle("OSD Version:", fyne.TextAlignLeading, versionStyle),
			a.versionLabel,
			layout.NewSpacer(),
			a.flashFirmwareButton,
		),
	))
	return a
}

func (a *App) setInfo(info *frskyosd.InfoMessage) {
	var text string
	if info != nil {
		if info.IsBootloader {
			a.uploadFontButton.Disable()
			text = "Bootloader"
		} else {
			text = fmt.Sprintf("%v.%v.%v", info.Version.Major, info.Version.Minor, info.Version.Patch)
			a.uploadFontButton.Enable()
		}
		a.flashFirmwareButton.Enable()
	} else {
		text = "Disconnected"
		a.uploadFontButton.Disable()
		a.flashFirmwareButton.Disable()
	}
	a.versionLabel.SetText(text)
}

func (a *App) connectOrDisconnect() {
	if a.connected {
		a.connectButton.SetText("Connect")
		a.connected = false
		a.clearFontItems()
		a.osd.Close()
		a.osd = nil
		a.setInfo(nil)
	} else {
		prog := dialog.NewProgressInfinite("Connecting...", "", a.window)
		go func() {
			osd, err := frskyosd.New(a.portsSelect.Selected)
			if err != nil {
				prog.Hide()
				a.showError(err)
				return
			}
			info, err := osd.Info()
			if err != nil {
				osd.Close()
				prog.Hide()
				a.showError(err)
				return
			}
			a.setInfo(info)
			a.osd = osd
			if info.IsBootloader {
				a.clearFontItems()
			} else {
				err = a.updateFontItems(func(p int) {
					prog.UpdateMessage(fmt.Sprintf("Reading font (%03d/%03d)...", p+1, len(a.fontItems)))
				})
				if err != nil {
					prog.Hide()
					a.showError(err)
					return
				}
			}
			a.connectedPort = a.portsSelect.Selected
			a.connectButton.SetText("Disconnect")
			a.connected = true
			prog.Hide()
		}()

		prog.Show()
	}
}

func (a *App) updatePorts() {
	a.ports = a.availablePorts()
}

func (a *App) updatePortsSelect() {
	for range time.Tick(time.Second) {
		a.updatePorts()
		found := false
		selected := a.portsSelect.Selected
		for _, v := range a.ports {
			if v == selected {
				found = true
				break
			}
		}
		a.portsSelect.Options = a.ports
		a.portsSelect.Refresh()
		if !found {
			if a.connected {
				a.connectOrDisconnect()
			}
			selected = ""
		}
		a.portsSelect.Selected = selected
		if a.portsSelect.OnChanged != nil {
			a.portsSelect.OnChanged(selected)
		}
		a.portsSelect.Refresh()
	}
}

func (a *App) availablePorts() []string {
	ports, err := frskyosd.AvailablePorts()
	if err != nil {
		a.showError(err)
		return nil
	}
	return ports
}

func (a *App) clearFontItems() {
	for _, v := range a.fontItems {
		v.SetFont(nil)
	}
}

func (a *App) storagePath(rel string) string {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	return filepath.Join(usr.HomeDir, ".frskyosd", filepath.FromSlash(rel))
}

func (a *App) updateRemoteFonts() {
	for _, origin := range fonts.Origins() {
		fonts, err := origin.Fonts()
		if err != nil {
			log.Printf("error getting fonts from %s: %v", origin.Name(), err)
			continue
		}
		for _, f := range fonts {
			r, err := f.Open()
			if err != nil {
				log.Printf("error opening font %s from %s: %v", f.Name, origin.Name(), err)
				continue
			}
			data, err := ioutil.ReadAll(r)
			if err != nil {
				log.Printf("error reading font %s from %s: %v", f.Name, origin.Name(), err)
				continue
			}
			dec, err := mcm.NewDecoder(bytes.NewReader(data))
			if err != nil {
				log.Printf("error decoding font %s from %s: %v", f.Name, origin.Name(), err)
				continue
			}
			if dec.NChars() == 0 {
				continue
			}
			p := a.storagePath(path.Join(fontsDir, origin.Name(), f.Name+fontsExt))
			if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
				continue
			}
			ioutil.WriteFile(p, data, 0644)
		}
	}
}

func (a *App) updateFontItems(progress func(int)) error {
	for ii, v := range a.fontItems {
		msg, err := a.osd.ReadFontChar(uint(ii))
		if err != nil {
			a.clearFontItems()
			return err
		}
		if progress != nil {
			progress(ii)
		}
		v.SetFont(msg)
	}
	return nil
}

func (a *App) portSelectionChanged(selected string) {
	if a.connected {
		if a.portsSelect.Selected != a.connectedPort {
			a.portsSelect.SetSelected(a.connectedPort)
		}
	} else {
		if selected != "" {
			a.connectButton.Enable()
		} else {
			a.connectButton.Disable()
		}
	}
}

func (a *App) uploadFont() {
	var tabItems []*widget.TabItem
	fontsStorageDir := a.storagePath(fontsDir)
	entries, _ := ioutil.ReadDir(fontsStorageDir)
	for _, entry := range entries {
		name := entry.Name()
		var fontItems []fyne.CanvasObject
		p := filepath.Join(fontsStorageDir, name)
		fontEntries, _ := ioutil.ReadDir(p)
		for _, fe := range fontEntries {
			if fe.IsDir() {
				continue
			}
			name := fe.Name()
			ext := filepath.Ext(name)
			if ext != fontsExt {
				continue
			}
			fontItems = append(fontItems, widget.NewButton(name[:len(name)-len(ext)], func() {
				a.uploadFontFilename(filepath.Join(p, name))
			}))
		}
		if len(fontItems) > 0 {
			sort.Slice(fontItems, func(i, j int) bool {
				wi := fontItems[i].(*widget.Button)
				wj := fontItems[j].(*widget.Button)
				if wi.Text == "Default" {
					return true
				}
				if wj.Text == "Default" {
					return false
				}
				return wi.Text < wj.Text
			})
			tabContent := widget.NewVBox(fontItems...)
			tabItems = append(tabItems, widget.NewTabItem(name, tabContent))
		}
	}
	if len(tabItems) > 0 {
		sort.Slice(tabItems, func(i, j int) bool {
			ii := tabItems[i]
			ij := tabItems[j]
			if ii.Text == "INAV" {
				return true
			}
			if ij.Text == "INAV" {
				return false
			}
			return ii.Text < ij.Text
		})
		tabs := widget.NewTabContainer(tabItems...)
		content := widget.NewHBox(tabs, layout.NewSpacer(), widget.NewVBox(widget.NewButton("Load File", a.uploadFontFileDialog)))
		a.uploadFontDialog = dialog.ShowCustom("Select Font", "Cancel", content, a.window)
	} else {
		a.uploadFontFileDialog()
	}
}

func (a *App) uploadFontData(r io.Reader) {
	prog := dialog.NewProgressInfinite("Uploading Font...", "", a.window)
	prog.Show()
	err := a.osd.UploadFont(r, func(done, total int) {
		prog.UpdateMessage(fmt.Sprintf("Writing font (%03d/%03d)...", done, total))
	})
	if err != nil {
		prog.Hide()
		a.showError(err)
		return
	}
	err = a.updateFontItems(func(p int) {
		prog.UpdateMessage(fmt.Sprintf("Reading font (%03d/%03d)...", p+1, len(a.fontItems)))
	})
	if err != nil {
		prog.Hide()
		a.showError(err)
		return
	}
	prog.Hide()
}

func (a *App) uploadFontFilename(filename string) {
	if a.uploadFontDialog != nil {
		a.uploadFontDialog.Hide()
		a.uploadFontDialog = nil
	}
	f, err := os.Open(filename)
	if err != nil {
		a.showError(err)
		return
	}
	defer f.Close()
	a.uploadFontData(f)
}

func (a *App) uploadFontFileDialog() {
	filename, err := dlgs.File().Filter("Font (*.mcm)", "mcm").Load()
	platformAfterFileDialog()
	if err != nil {
		if err != dlgs.ErrCancelled {
			a.showError(err)
		}
		return
	}
	a.uploadFontFilename(filename)
}

func (a *App) flashFirmware(r io.Reader) {
	prog := dialog.NewProgressInfinite("Flashing...", "", a.window)
	prog.Show()
	err := a.osd.FlashFirmware(r, func(done int, total int) {
		prog.UpdateMessage(fmt.Sprintf("%03d/%03d bytes written...", done, total))
	})
	prog.Hide()
	if err != nil {
		a.showError(err)
		return
	}
	info, err := a.osd.Info()
	if err != nil {
		a.showError(err)
		return
	}
	a.setInfo(info)
}

func (a *App) selectFirmware() {
	a.selectFirmwareFile()
}

func (a *App) selectFirmwareFile() {
	filename, err := dlgs.File().Filter("Firmware (*.bin)", "bin").Load()
	platformAfterFileDialog()
	if err != nil {
		if err != dlgs.ErrCancelled {
			a.showError(err)
		}
		return
	}
	f, err := os.Open(filename)
	if err != nil {
		a.showError(err)
		return
	}
	defer f.Close()
	a.flashFirmware(f)

}

func (a *App) showError(err error) {
	if a.window != nil {
		dialog.ShowError(err, a.window)
	}
	log.Println(err)
}

func (a *App) startAutoupdater() {
	src, err := autoupdater.NewSource(updatesSource)
	if err != nil {
		log.Errorln(err)
		return
	}
	au, err := autoupdater.New(&autoupdater.Options{
		Version:         appVersion,
		AcceptPreleases: false,
		NoSkipRelease:   true,
		Source:          src,
		Dialog:          a,
	})
	if err != nil {
		log.Errorln(err)
		return
	}
	au.ScheduleCheckingForUpdates(12 * time.Hour)
}

// ShowUpdaterDialog implements the autoupdater.Dialog interface
func (a *App) ShowUpdaterDialog(opts *autoupdater.DialogOptions) {
	var resp autoupdater.DialogResponse
	var msg string
	if opts.AllowsResponse(autoupdater.DialogResponseDownloadAndInstall) {
		resp = autoupdater.DialogResponseDownloadAndInstall
		msg = fmt.Sprintf("Would you like to download and install it?")
	} else if opts.AllowsResponse(autoupdater.DialogResponseDownload) {
		resp = autoupdater.DialogResponseDownload
		msg = fmt.Sprintf("Would you like to download it?")
	} else {
		return
	}
	title := fmt.Sprintf("Version %s is available", opts.AvailableRelease.Version)
	confirmDialog := dialog.NewConfirm(title, msg, func(ok bool) {
		if ok {
			opts.Response(resp)
		} else {
			opts.Response(autoupdater.DialogResponseCancel)
		}
	}, a.window)
	confirmDialog.Show()
}

// Run starts the app
func (a *App) Run() {
	a.setInfo(nil)
	go a.updatePortsSelect()
	go func() {
		a.updateRemoteFonts()
		for range time.Tick(1 * time.Hour) {
			a.updateRemoteFonts()
		}
	}()
	a.window.Resize(fyne.NewSize(516, 475))
	a.window.SetFixedSize(true)
	go a.startAutoupdater()
	a.window.ShowAndRun()
}

const (
	fontCharCount = mcm.ExtendedCharNum
	fontRowSize   = 32
)

func main() {
	debug := flag.Bool("debug", false, "Set logging level to debug")
	trace := flag.Bool("trace", false, "Set logging level to trace. Implies debug.")
	flag.Parse()
	if *trace || os.Getenv("FRSKY_OSD_TRACE") != "" {
		log.SetLevel(log.TraceLevel)
	} else if *debug || os.Getenv("FRSKY_OSD_DEBUG") != "" {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	platformInit()
	app := newApp()
	app.Run()
}
