package main // import "osdapp"

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"

	"github.com/fiam/max7456tool/mcm"
	log "github.com/sirupsen/logrus"
	dlgs "github.com/sqweek/dialog"

	"osdapp/firmware"
	"osdapp/fonts"
	"osdapp/frskyosd"
	"osdapp/internal/autoupdater"
	"osdapp/internal/dialog"
	"osdapp/internal/osdversion"
)

const (
	fontsDir   = "fonts"
	fontsExt   = ".mcm"
	appVersion = "1.0.1"

	updatesSource = "https://github.com/FrSkyRC/FrSkyOSDApp"

	settingsNotSupportedMessage = "Settings require firmware v2.\nUse the \"Flash Firmware\" button to upgrade."
	bootloaderModeMessage       = `That probably means there's no valid firmware loaded.
Use the "Flash Firmware" button to flash a firmware`
	fontsUpdateInterval = 1 * time.Hour
)

// App is an opaque type that contains the whole application state
type App struct {
	app                  fyne.App
	connected            bool
	ports                []string
	window               fyne.Window
	portsSelect          *widget.Select
	connectButton        *widget.Button
	versionLabel         *widget.Label
	uploadFontButton     *widget.Button
	fontItems            []*FontIcon
	uploadFontDialog     dialog.Dialog
	settingsButton       *widget.Button
	flashFirmwareButton  *widget.Button
	selectFirmwareDialog *firmwaresDialog
	connectedPort        string
	osd                  *frskyosd.OSD
	info                 *frskyosd.InfoMessage
}

func newApp() *App {
	a := &App{}
	a.app = app.New()
	a.updatePorts()
	a.connectButton = widget.NewButton("Connect", a.connectOrDisconnect)
	a.connectButton.Disable()
	a.portsSelect = widget.NewSelect(a.ports, a.portSelectionChanged)
	a.uploadFontButton = widget.NewButton("Upload Font", a.uploadFont)
	a.settingsButton = widget.NewButton("Settings", a.showSettings)
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
		widget.NewHBox(a.settingsButton, layout.NewSpacer()),
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
			text = osdversion.Format(int(info.Version.Major), int(info.Version.Minor), int(info.Version.Patch))
			a.uploadFontButton.Enable()
			a.settingsButton.Enable()
		}
		a.flashFirmwareButton.Enable()
	} else {
		text = "Disconnected"
		a.uploadFontButton.Disable()
		a.flashFirmwareButton.Disable()
		a.settingsButton.Disable()
	}
	a.versionLabel.SetText(text)
	a.info = info
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
			if info.IsBootloader {
				dlg := dialog.ShowInformation("OSD is in bootloader mode", bootloaderModeMessage, a.window)
				dlg.Show()
			}
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

func (a *App) isConnectedToV2() bool {
	if a.info != nil {
		if a.info.Version.Major >= 2 {
			return true
		}
		return a.info.Version.Major == 1 && a.info.Version.Minor >= 99
	}
	return false
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

func (a *App) timestampPath(name string) string {
	return a.storagePath(path.Join("timestamps", name))
}

func (a *App) timestampHasElapsed(name string, interval time.Duration) bool {
	p := a.timestampPath(name)
	data, err := ioutil.ReadFile(p)
	if err == nil {
		n, err := strconv.ParseInt(string(data), 10, 64)
		if err == nil {
			if time.Unix(n, 0).Add(interval).After(time.Now()) {
				return false
			}
		}
	}
	return true
}

func (a *App) updateTimestamp(name string) {
	p := a.timestampPath(name)
	data := []byte(strconv.FormatInt(time.Now().Unix(), 10))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err == nil {
		ioutil.WriteFile(p, data, 0644)
	}

}

func (a *App) updateRemoteFonts() {
	const (
		timestampName = "fonts"
	)
	if !a.timestampHasElapsed(timestampName, fontsUpdateInterval) {
		return
	}
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
	a.updateTimestamp(timestampName)
}

func (a *App) updateFontItems(progress func(int)) error {
	// Used for testing
	if os.Getenv("FRSKY_OSD_SKIP_FONT_ITEMS") == "1" {
		return nil
	}
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
		title := "No fonts could be automatically downloaded"
		msg := "Please, check your connectivity to github.com\nor select a font manually."
		warn := dialog.NewInformation(title, msg, a.window)
		warn.SetCallback(func(_ bool) {
			a.uploadFontFileDialog()
		})
		warn.Show()
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

func (a *App) showFirmwareLoadingError(err error) {
	msg := fmt.Sprintf("%v\nPlease, select a file manually", err)
	dlg := dialog.ShowInformation("No firmwares could be loaded", msg, a.window)
	dlg.SetCallback(func(_ bool) {
		a.selectFirmwareFile()
	})
	dlg.Show()
}

func (a *App) selectFirmware() {
	progress := dialog.NewProgressInfinite("Updating", "Check for firmware updates", a.window)
	progress.Show()
	go func() {
		firmwares, err := firmware.Load()
		if err != nil {
			progress.Hide()
			a.showFirmwareLoadingError(err)
			return
		}
		if len(firmwares) == 0 {
			progress.Hide()
			a.showFirmwareLoadingError(errors.New("no firmwares found"))
			return
		}
		a.selectFirmwareDialog = newFirmwaresDialog(firmwares, a.window)
		a.selectFirmwareDialog.OnFirmwareSelected = func(f *firmware.Firmware) {
			a.selectFirmwareEntry(f)
		}
		a.selectFirmwareDialog.OnSelectFile = a.selectFirmwareFile
		progress.Hide()
		a.selectFirmwareDialog.Show()
	}()
}

func (a *App) selectFirmwareEntry(f *firmware.Firmware) {
	resp, err := http.Get(f.URL)
	if err != nil {
		a.showError(err)
		return
	}
	defer resp.Body.Close()
	a.flashFirmware(resp.Body)
}

func (a *App) selectFirmwareFile() {
	if a.selectFirmwareDialog != nil {
		a.selectFirmwareDialog.Hide()
		a.selectFirmwareDialog = nil
	}
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

func (a *App) drawCorner(x, y, dx, dy int) error {
	if err := a.osd.SetStrokeColor(frskyosd.CWhite); err != nil {
		return err
	}
	if err := a.osd.MoveToPoint(x, y); err != nil {
		return err
	}
	if err := a.osd.StrokeLineToPoint(x+dx, y); err != nil {
		return err
	}
	if err := a.osd.MoveToPoint(x, y); err != nil {
		return err
	}
	if err := a.osd.StrokeLineToPoint(x, y+dy); err != nil {
		return err
	}
	ox := 1
	if dx < 0 {
		ox = -1
	}
	oy := 1
	if dy < 0 {
		oy = -1
	}
	for ii := x; ii != x+dx; ii += ox {
		if ii%5 == 0 {
			if err := a.osd.MoveToPoint(ii, y); err != nil {
				return err
			}
			if err := a.osd.StrokeLineToPoint(ii, y+oy*5); err != nil {
				return err
			}
		}
	}
	for ii := y; ii != y+dy; ii += oy {
		if ii%5 == 0 {
			if err := a.osd.MoveToPoint(x, ii); err != nil {
				return err
			}
			if err := a.osd.StrokeLineToPoint(x+ox*5, ii); err != nil {
				return err
			}
		}
	}
	if err := a.osd.MoveToPoint(x, y); err != nil {
		return err
	}
	if err := a.osd.SetStrokeColor(frskyosd.CBlack); err != nil {
		return err
	}
	if err := a.osd.MoveToPoint(x+ox, y+oy); err != nil {
		return err
	}
	if err := a.osd.StrokeLineToPoint(x+dx, y+oy); err != nil {
		return err
	}
	if err := a.osd.MoveToPoint(x+ox, y+oy); err != nil {
		return err
	}
	if err := a.osd.StrokeLineToPoint(x+ox, y+dy); err != nil {
		return err
	}

	if err := a.osd.SetStrokeColor(frskyosd.CWhite); err != nil {
		return err
	}
	ox *= 2
	oy *= 2
	if err := a.osd.MoveToPoint(x+ox, y+oy); err != nil {
		return err
	}
	if err := a.osd.StrokeLineToPoint(x+dx, y+oy); err != nil {
		return err
	}
	if err := a.osd.MoveToPoint(x+ox, y+oy); err != nil {
		return err
	}
	if err := a.osd.StrokeLineToPoint(x+ox, y+dy); err != nil {
		return err
	}
	return nil
}

// Show some stuff on the screen so the user
// can get a better idea of how the changes are
// affecting the OSD
func (a *App) prepareToShowSettings() error {
	const (
		rectSize   = 50
		rectMargin = 20
		lineSize   = 50
	)
	info, err := a.osd.Info()
	if err != nil {
		return err
	}
	if err := a.osd.TransactionBegin(); err != nil {
		return err
	}
	if err := a.osd.ResetDrawing(); err != nil {
		return err
	}
	if err := a.osd.ClearScreen(); err != nil {
		return err
	}
	midX := int(info.Pixels.Width / 2)
	midY := int(info.Pixels.Height / 2)
	rectY := midY - rectSize/2
	if err := a.osd.SetFillColor(frskyosd.CBlack); err != nil {
		return err
	}
	if err := a.osd.FillRect(midX-rectSize/2-rectMargin-rectSize,
		rectY, rectSize, rectSize); err != nil {
		return err
	}
	if err := a.osd.SetFillColor(frskyosd.CGray); err != nil {
		return err
	}
	if err := a.osd.FillRect(midX-rectSize/2, rectY, rectSize, rectSize); err != nil {
		return err
	}
	if err := a.osd.SetFillColor(frskyosd.CWhite); err != nil {
		return err
	}
	if err := a.osd.FillRect(midX+rectSize/2+rectMargin, rectY, rectSize, rectSize); err != nil {
		return err
	}
	lineX := int(info.Pixels.Width) - 1
	lineY := int(info.Pixels.Height) - 1
	if err := a.drawCorner(0, 0, lineSize, lineSize); err != nil {
		return err
	}
	if err := a.drawCorner(0, lineY, lineSize, -lineSize); err != nil {
		return err
	}
	if err := a.drawCorner(lineX, 0, -lineSize, lineSize); err != nil {
		return err
	}
	if err := a.drawCorner(lineX, lineY, -lineSize, -lineSize); err != nil {
		return err
	}
	return a.osd.TransactionCommit()
}

func (a *App) showSettings() {
	if a.window == nil {
		return
	}
	if !a.isConnectedToV2() {
		if a.window != nil {
			dialog.ShowInformation("Version 2 required", settingsNotSupportedMessage, a.window)
		}
		return
	}
	a.settingsButton.Disable()
	loading := dialog.NewProgressInfinite("Loading", "", a.window)
	loading.Show()
	go func() {
		settings, err := a.osd.ReadSettings()
		a.settingsButton.Enable()
		if err != nil {
			loading.Hide()
			a.showError(err)
			return
		}
		if err := a.prepareToShowSettings(); err != nil {
			loading.Hide()
			a.showError(err)
			return
		}
		loading.Hide()
		settingsDialog := newSettingsDialog(a.osd, settings, a.window)
		settingsDialog.OnChanged = func(s *frskyosd.SettingsMessage) {
			go func() {
				_, err := a.osd.SetSettings(s)
				if err != nil {
					a.showError(err)
				}
			}()
		}
		settingsDialog.OnClosed = func(changed bool) {
			go func() {
				if changed {
					if err := a.osd.SaveSettings(); err != nil {
						a.showError(err)
					}
				}
				a.osd.ClearScreen()
			}()
		}
	}()
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
		for range time.Tick(fontsUpdateInterval) {
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
