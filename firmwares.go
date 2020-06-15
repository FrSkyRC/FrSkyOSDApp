package main

import (
	"fmt"

	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"

	"github.com/pkg/browser"

	"osdapp/firmware"
)

const (
	firmwaresDialogWidth = 350
)

type firmwaresDialog struct {
	win                *widget.PopUp
	bg                 *canvas.Rectangle
	title              *widget.Label
	firmwares          *widget.Box
	buttons            *widget.Box
	content            *fyne.Container
	OnFirmwareSelected func(f *firmware.Firmware)
	OnSelectFile       func()
}

func newFirmwaresDialog(firmwares []*firmware.Firmware, parent fyne.Window) *firmwaresDialog {
	d := &firmwaresDialog{}
	d.bg = canvas.NewRectangle(theme.BackgroundColor())
	d.bg.FillColor = theme.BackgroundColor()
	d.bg.StrokeColor = theme.TextColor()
	d.title = widget.NewLabelWithStyle("Select Firmware", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	var selectEntries []fyne.CanvasObject
	var infoEntries []fyne.CanvasObject
	for _, f := range firmwares {
		f := f
		var name string
		if vers, err := f.VersionName(); err == nil {
			if date, err := f.Date(); err == nil {
				name = fmt.Sprintf("%s (%s)", vers, date.Format("02 Jan 2006"))
			}
		}
		if name == "" {
			if n, err := f.Filename(); err == nil {
				name = n
			} else {
				name = f.URL
			}
		}
		selectFirmware := widget.NewButton(name, func() {
			d.OnFirmwareSelected(f)
		})
		selectEntries = append(selectEntries, selectFirmware)
		info := widget.NewButtonWithIcon("", theme.InfoIcon(), func() {
			browser.OpenURL(f.ReleaseNotesURL)
		})
		infoEntries = append(infoEntries, info)
	}
	d.firmwares = widget.NewHBox(widget.NewVBox(selectEntries...), widget.NewVBox(infoEntries...))
	selectFileButton := widget.NewButtonWithIcon("Select a File", theme.FolderOpenIcon(), func() {
		d.OnSelectFile()
	})
	cancelButton := widget.NewButtonWithIcon("Close", theme.CancelIcon(), d.Hide)
	d.buttons = widget.NewHBox(selectFileButton, layout.NewSpacer(), cancelButton)
	d.content = fyne.NewContainerWithLayout(d,
		d.bg,
		d.title,
		d.firmwares,
		d.buttons,
	)
	d.win = widget.NewModalPopUp(d.content, parent.Canvas())
	return d
}

func (d *firmwaresDialog) SetDismissText(label string) {}
func (d *firmwaresDialog) SetCallback(func(bool))      {}

func (d *firmwaresDialog) Show() {
	d.win.Show()
}

func (d *firmwaresDialog) Hide() {
	d.win.Hide()
}

func (d *firmwaresDialog) Layout(obj []fyne.CanvasObject, size fyne.Size) {
	d.bg.Move(fyne.NewPos(-theme.Padding(), -theme.Padding()))
	d.bg.Resize(size.Add(fyne.NewSize(theme.Padding()*2, theme.Padding()*2)))
	titleSize := d.title.MinSize()
	d.title.Move(fyne.NewPos((size.Width-titleSize.Width)/2, theme.Padding()))

	firmwaresSize := d.firmwares.MinSize()
	d.firmwares.Move(fyne.NewPos((size.Width-firmwaresSize.Width)/2, titleSize.Height+theme.Padding()*2))

	buttonsSize := d.buttons.MinSize()
	d.buttons.Resize(fyne.NewSize(size.Width, buttonsSize.Height))
	d.buttons.Move(fyne.NewPos(theme.Padding(), size.Height-buttonsSize.Height))
}

func (d *firmwaresDialog) MinSize(obj []fyne.CanvasObject) fyne.Size {
	titleMinSize := d.title.MinSize()
	firmwaresMinSize := d.firmwares.MinSize()
	buttonsMinSize := d.buttons.MinSize()
	minHeight := titleMinSize.Height + firmwaresMinSize.Height + buttonsMinSize.Height + theme.Padding()*5
	return fyne.NewSize(firmwaresDialogWidth, minHeight+30)
}
