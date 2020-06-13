package main

import (
	"image/color"
	"osdapp/frskyosd"
	"strconv"

	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
)

const (
	settingsDialogWidth  = 400
	settingsDialogHeight = 300

	settingMinValue = -128
	settingMaxValue = 127
)

type settingBox struct {
	bg        *canvas.Rectangle
	title     *widget.Label
	value     *widget.Label
	slider    *widget.Slider
	content   *fyne.Container
	plus      *widget.Button
	minus     *widget.Button
	val       int
	OnChanged func(sb *settingBox)
}

func newSettingBox(name string) *settingBox {
	sb := &settingBox{}
	sb.bg = canvas.NewRectangle(theme.BackgroundColor())
	sb.bg.FillColor = color.NRGBA{R: uint8(0), G: uint8(0), B: uint8(0), A: 127}
	sb.bg.StrokeColor = theme.TextColor()
	sb.title = widget.NewLabel(name)
	sb.value = widget.NewLabelWithStyle("0", fyne.TextAlignCenter, fyne.TextStyle{Monospace: true})
	sb.slider = widget.NewSlider(settingMinValue, settingMaxValue)
	sb.slider.Step = 1
	sb.slider.OnChanged = sb.sliderChanged
	sb.plus = widget.NewButtonWithIcon("", theme.ContentAddIcon(), sb.increaseVal)
	sb.minus = widget.NewButtonWithIcon("", theme.ContentRemoveIcon(), sb.decreaseVal)
	sb.content = fyne.NewContainerWithLayout(sb,
		sb.bg,
		sb.title,
		sb.slider,
		sb.value,
		sb.plus,
		sb.minus,
	)
	sb.UpdateVal(sb.val)
	return sb
}

func (sb *settingBox) Content() fyne.CanvasObject {
	return sb.content
}

func (sb *settingBox) Layout(obj []fyne.CanvasObject, size fyne.Size) {
	const titleWidth = 125
	const valueWidth = 36
	const buttonSize = 12
	titleSize := sb.title.MinSize()
	sb.title.Move(fyne.NewPos(0, (size.Height-titleSize.Height)/2))
	sb.title.Resize(fyne.NewSize(titleWidth, titleSize.Height))

	sliderSize := sb.slider.MinSize()
	sb.slider.Resize(fyne.NewSize(size.Width-titleWidth-theme.Padding()*2-valueWidth-buttonSize, sliderSize.Height))
	sb.slider.Move(fyne.NewPos(titleWidth+theme.Padding(), (size.Height-sliderSize.Height)/2))

	valueSize := sb.value.MinSize()
	sb.value.Resize(fyne.NewSize(valueWidth, valueSize.Height))
	valueX := size.Width - valueWidth - buttonSize - theme.Padding()
	sb.value.Move(fyne.NewPos(valueX, (size.Height-valueSize.Height)/2))

	buttonX := size.Width - buttonSize
	sb.plus.Resize(fyne.NewSize(buttonSize, buttonSize))
	sb.plus.Move(fyne.NewPos(buttonX, size.Height/2-buttonSize-1))
	sb.minus.Resize(fyne.NewSize(buttonSize, buttonSize))
	sb.minus.Move(fyne.NewPos(buttonX, size.Height/2+1))

	bgHeight := buttonSize*2 + 4
	sb.bg.Resize(fyne.NewSize(valueWidth, bgHeight))
	sb.bg.Move(fyne.NewPos(valueX, (size.Height-bgHeight)/2))
}

func (sb *settingBox) MinSize(obj []fyne.CanvasObject) fyne.Size {
	titleSize := sb.title.MinSize()
	return fyne.NewSize(sb.minWidth(), titleSize.Height)
}

func (sb *settingBox) minWidth() int {
	return settingsDialogWidth - theme.Padding()*2
}

func (sb *settingBox) sliderChanged(val float64) {
	sb.updateValAndNotify(int(val))
}

func (sb *settingBox) UpdateVal(val int) {
	sb.val = val
	sb.value.SetText(strconv.Itoa(val))
	sb.slider.Value = float64(val)
	sb.slider.Refresh()
	if val < settingMaxValue {
		sb.plus.Enable()
	} else {
		sb.plus.Disable()
	}
	if val > settingMinValue {
		sb.minus.Enable()
	} else {
		sb.minus.Disable()
	}
}

func (sb *settingBox) updateValAndNotify(val int) {
	sb.UpdateVal(val)
	onChanged := sb.OnChanged
	if onChanged != nil {
		onChanged(sb)
	}
}

func (sb *settingBox) Val() int {
	return sb.val
}

func (sb *settingBox) increaseVal() {
	if sb.val < settingMaxValue {
		sb.updateValAndNotify(sb.val + 1)
	}
}

func (sb *settingBox) decreaseVal() {
	if sb.val > settingMinValue {
		sb.updateValAndNotify(sb.val - 1)
	}
}

type settingsDialog struct {
	settings          *frskyosd.SettingsMessage
	win               *widget.PopUp
	bg                *canvas.Rectangle
	brightness        *settingBox
	horizontalOffset  *settingBox
	verticalOffset    *settingBox
	content           *fyne.Container
	label             fyne.CanvasObject
	settingsContainer fyne.CanvasObject
	restoreButton     *widget.Button
	closeButton       *widget.Button
	save              *widget.Button
	buttons           *widget.Box
	parent            fyne.Window
	closed            bool
	OnChanged         func(settings *frskyosd.SettingsMessage)
	OnClosed          func()
}

func newSettingsDialog(settings *frskyosd.SettingsMessage, parent fyne.Window) *settingsDialog {
	d := &settingsDialog{settings: settings}
	d.bg = canvas.NewRectangle(theme.BackgroundColor())
	d.label = widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	d.brightness = newSettingBox("Brightness:")
	d.brightness.UpdateVal(int(settings.Brightness))
	d.brightness.OnChanged = d.onSettingChanged
	d.horizontalOffset = newSettingBox("Horizontal Offset:")
	d.horizontalOffset.UpdateVal(int(settings.HorizontalOffset))
	d.horizontalOffset.OnChanged = d.onSettingChanged
	d.verticalOffset = newSettingBox("Vertical Offset:")
	d.verticalOffset.UpdateVal(int(settings.VerticalOffset))
	d.verticalOffset.OnChanged = d.onSettingChanged
	d.settingsContainer = widget.NewVBox(
		d.brightness.Content(),
		d.horizontalOffset.Content(),
		d.verticalOffset.Content(),
	)
	d.restoreButton = widget.NewButtonWithIcon("Restore Defaults", theme.DeleteIcon(), d.restore)
	d.closeButton = widget.NewButtonWithIcon("Close", theme.CancelIcon(), d.dismiss)
	d.buttons = widget.NewHBox(
		d.restoreButton,
		layout.NewSpacer(),
		d.closeButton,
	)
	d.content = fyne.NewContainerWithLayout(d,
		d.bg,
		d.label,
		d.settingsContainer,
		d.buttons,
	)
	d.win = widget.NewModalPopUp(d.content, parent.Canvas())
	d.applyTheme()
	d.win.Show()
	return d
}

func (d *settingsDialog) Layout(obj []fyne.CanvasObject, size fyne.Size) {
	d.bg.Move(fyne.NewPos(-theme.Padding(), -theme.Padding()))
	d.bg.Resize(size.Add(fyne.NewSize(theme.Padding()*2, theme.Padding()*2)))
	titleSize := d.label.MinSize()
	d.label.Move(fyne.NewPos((settingsDialogWidth-titleSize.Width)/2, theme.Padding()))

	settingsSize := d.settingsContainer.MinSize()
	d.settingsContainer.Resize(fyne.NewSize(settingsDialogWidth, settingsSize.Height))
	d.settingsContainer.Move(fyne.NewPos(0, titleSize.Height+theme.Padding()))

	buttonsSize := d.buttons.MinSize()
	d.buttons.Resize(fyne.NewSize(settingsDialogWidth, buttonsSize.Height))
	d.buttons.Move(fyne.NewPos(0, settingsDialogHeight-buttonsSize.Height))
}

func (d *settingsDialog) MinSize(obj []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(settingsDialogWidth, settingsDialogHeight)
}

func (d *settingsDialog) applyTheme() {
	r, g, b, _ := theme.BackgroundColor().RGBA()
	bg := &color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 230}
	d.bg.FillColor = bg
}

func (d *settingsDialog) onChanged() {
	onChanged := d.OnChanged
	if onChanged != nil {
		onChanged(d.settings)
	}
}

func (d *settingsDialog) onSettingChanged(sb *settingBox) {
	switch sb {
	case d.brightness:
		d.settings.Brightness = int8(sb.Val())
	case d.horizontalOffset:
		d.settings.HorizontalOffset = int8(sb.Val())
	case d.verticalOffset:
		d.settings.VerticalOffset = int8(sb.Val())
	default:
		panic("invalid setting")
	}
	d.onChanged()
}

func (d *settingsDialog) restore() {
	d.settings.RestoreDefaults()
	d.brightness.UpdateVal(int(d.settings.Brightness))
	d.horizontalOffset.UpdateVal(int(d.settings.HorizontalOffset))
	d.verticalOffset.UpdateVal(int(d.settings.VerticalOffset))
	d.settings.Brightness = 0
	d.onChanged()
}

func (d *settingsDialog) dismiss() {
	d.closed = true
	d.win.Hide()
	onClosed := d.OnClosed
	if onClosed != nil {
		onClosed()
	}
}