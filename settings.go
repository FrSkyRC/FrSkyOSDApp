package main

import (
	"fmt"
	"image/color"
	"os"
	"osdapp/frskyosd"
	"strconv"
	"time"

	"fyne.io/fyne"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
)

const (
	settingsDialogWidth  = 450
	settingsDialogHeight = 300

	settingMinValue = -128
	settingMaxValue = 127

	helpText = `Use your goggles or screen to view
the effect of the the OSD settings
while you adjust them.

Changes are saved to the OSD, but the
defaults can be restored at any time.`
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
	osd                   *frskyosd.OSD
	settings              *frskyosd.SettingsMessage
	win                   *widget.PopUp
	bg                    *canvas.Rectangle
	titleLabel            *widget.Label
	awaitingLabel         *widget.Label
	awaitingSubtitleLabel *widget.Label
	helpLabel             *widget.Label
	progress              *widget.ProgressBarInfinite
	brightness            *settingBox
	horizontalOffset      *settingBox
	verticalOffset        *settingBox
	content               *fyne.Container
	settingsContainer     fyne.CanvasObject
	restoreButton         *widget.Button
	closeButton           *widget.Button
	save                  *widget.Button
	buttons               *widget.Box
	parent                fyne.Window
	closed                bool
	changed               bool
	OnChanged             func(settings *frskyosd.SettingsMessage)
	OnClosed              func(changed bool)
}

func newSettingsDialog(osd *frskyosd.OSD, settings *frskyosd.SettingsMessage, parent fyne.Window) *settingsDialog {
	d := &settingsDialog{osd: osd, settings: settings}
	d.bg = canvas.NewRectangle(theme.BackgroundColor())
	d.titleLabel = widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	d.awaitingLabel = widget.NewLabel("Awaiting for camera...")
	d.awaitingSubtitleLabel = widget.NewLabel("Connect a camera to configure the OSD settings")
	d.progress = widget.NewProgressBarInfinite()
	d.helpLabel = widget.NewLabelWithStyle(helpText, fyne.TextAlignCenter, fyne.TextStyle{Monospace: true})
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
	d.restoreButton.Disable()
	d.closeButton = widget.NewButtonWithIcon("Close", theme.CancelIcon(), d.dismiss)
	d.buttons = widget.NewHBox(
		d.restoreButton,
		layout.NewSpacer(),
		d.closeButton,
	)
	d.content = fyne.NewContainerWithLayout(d,
		d.bg,
		d.titleLabel,
		d.awaitingLabel,
		d.progress,
		d.awaitingSubtitleLabel,
		d.settingsContainer,
		d.helpLabel,
		d.buttons,
	)
	cam, _ := d.osd.ActiveCamera()
	if cam > 0 {
		d.onCameraConnected()
	} else {
		d.onCameraDisconected()
		go d.awaitForCamera()
	}
	d.win = widget.NewModalPopUp(d.content, parent.Canvas())
	d.applyTheme()
	d.win.Show()
	return d
}

func (d *settingsDialog) Layout(obj []fyne.CanvasObject, size fyne.Size) {
	const (
		progressHeight = 40
	)
	d.bg.Move(fyne.NewPos(-theme.Padding(), -theme.Padding()))
	d.bg.Resize(size.Add(fyne.NewSize(theme.Padding()*2, theme.Padding()*2)))
	titleSize := d.titleLabel.MinSize()
	d.titleLabel.Move(fyne.NewPos((settingsDialogWidth-titleSize.Width)/2, theme.Padding()))

	settingsSize := d.settingsContainer.MinSize()
	d.settingsContainer.Resize(fyne.NewSize(settingsDialogWidth, settingsSize.Height))
	d.settingsContainer.Move(fyne.NewPos(0, titleSize.Height+theme.Padding()))

	d.helpLabel.Move(fyne.NewPos((settingsDialogWidth)/2,
		titleSize.Height+theme.Padding()*3+settingsSize.Height))

	d.progress.Resize(fyne.NewSize(settingsDialogWidth, progressHeight))
	d.progress.Move(fyne.NewPos(0, (size.Height-progressHeight)/2))

	awaitingSize := d.awaitingLabel.MinSize()
	d.awaitingLabel.Move(fyne.NewPos((size.Width-awaitingSize.Width)/2,
		size.Height/2-progressHeight/2-theme.Padding()-awaitingSize.Height))

	awaitingSubtitleSize := d.awaitingSubtitleLabel.MinSize()
	d.awaitingSubtitleLabel.Move(fyne.NewPos((size.Width-awaitingSubtitleSize.Width)/2,
		size.Height/2+progressHeight/2+theme.Padding()))

	buttonsSize := d.buttons.MinSize()
	d.buttons.Resize(fyne.NewSize(settingsDialogWidth, buttonsSize.Height))
	d.buttons.Move(fyne.NewPos(0, settingsDialogHeight-buttonsSize.Height))
}

func (d *settingsDialog) MinSize(obj []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(settingsDialogWidth, settingsDialogHeight)
}

func (d *settingsDialog) awaitForCamera() {
	for {
		cam, err := d.osd.ActiveCamera()
		if err != nil {
			d.dismiss()
		}
		if cam > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	d.onCameraConnected()
}

func (d *settingsDialog) onCameraDisconected() {
	d.progress.Show()
	d.awaitingLabel.Show()
	d.awaitingSubtitleLabel.Show()
	d.settingsContainer.Hide()
	d.helpLabel.Hide()
	d.restoreButton.Disable()
}

func (d *settingsDialog) onCameraConnected() {
	d.progress.Hide()
	d.awaitingLabel.Hide()
	d.awaitingSubtitleLabel.Hide()
	d.settingsContainer.Show()
	d.helpLabel.Show()
	d.restoreButton.Enable()
	if err := d.drawTestPattern(); err != nil {
		fmt.Fprintf(os.Stderr, "error drawing test pattern: %v\n", err)
	}
}

func (d *settingsDialog) drawCorner(x, y, dx, dy int) error {
	osd := d.osd
	if err := osd.SetStrokeColor(frskyosd.CWhite); err != nil {
		return err
	}
	if err := osd.MoveToPoint(x, y); err != nil {
		return err
	}
	if err := osd.StrokeLineToPoint(x+dx, y); err != nil {
		return err
	}
	if err := osd.MoveToPoint(x, y); err != nil {
		return err
	}
	if err := osd.StrokeLineToPoint(x, y+dy); err != nil {
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
			if err := osd.MoveToPoint(ii, y); err != nil {
				return err
			}
			if err := osd.StrokeLineToPoint(ii, y+oy*5); err != nil {
				return err
			}
		}
	}
	for ii := y; ii != y+dy; ii += oy {
		if ii%5 == 0 {
			if err := osd.MoveToPoint(x, ii); err != nil {
				return err
			}
			if err := osd.StrokeLineToPoint(x+ox*5, ii); err != nil {
				return err
			}
		}
	}
	if err := osd.MoveToPoint(x, y); err != nil {
		return err
	}
	if err := osd.SetStrokeColor(frskyosd.CBlack); err != nil {
		return err
	}
	if err := osd.MoveToPoint(x+ox, y+oy); err != nil {
		return err
	}
	if err := osd.StrokeLineToPoint(x+dx, y+oy); err != nil {
		return err
	}
	if err := osd.MoveToPoint(x+ox, y+oy); err != nil {
		return err
	}
	if err := osd.StrokeLineToPoint(x+ox, y+dy); err != nil {
		return err
	}

	if err := osd.SetStrokeColor(frskyosd.CWhite); err != nil {
		return err
	}
	ox *= 2
	oy *= 2
	if err := osd.MoveToPoint(x+ox, y+oy); err != nil {
		return err
	}
	if err := osd.StrokeLineToPoint(x+dx, y+oy); err != nil {
		return err
	}
	if err := osd.MoveToPoint(x+ox, y+oy); err != nil {
		return err
	}
	if err := osd.StrokeLineToPoint(x+ox, y+dy); err != nil {
		return err
	}
	return nil
}

// Show some stuff on the screen so the user
// can get a better idea of how the changes are
// affecting the OSD
func (d *settingsDialog) drawTestPattern() error {
	const (
		rectSize   = 50
		rectMargin = 20
		lineSize   = 50
	)
	osd := d.osd
	info, err := osd.Info()
	if err != nil {
		return err
	}
	if err := osd.TransactionBegin(); err != nil {
		return err
	}
	if err := osd.ResetDrawing(); err != nil {
		return err
	}
	if err := osd.ClearScreen(); err != nil {
		return err
	}
	midX := int(info.Pixels.Width / 2)
	midY := int(info.Pixels.Height / 2)
	rectY := midY - rectSize/2
	if err := osd.SetFillColor(frskyosd.CBlack); err != nil {
		return err
	}
	if err := osd.FillRect(midX-rectSize/2-rectMargin-rectSize,
		rectY, rectSize, rectSize); err != nil {
		return err
	}
	if err := osd.SetFillColor(frskyosd.CGray); err != nil {
		return err
	}
	if err := osd.FillRect(midX-rectSize/2, rectY, rectSize, rectSize); err != nil {
		return err
	}
	if err := osd.SetFillColor(frskyosd.CWhite); err != nil {
		return err
	}
	if err := osd.FillRect(midX+rectSize/2+rectMargin, rectY, rectSize, rectSize); err != nil {
		return err
	}
	lineX := int(info.Pixels.Width) - 1
	lineY := int(info.Pixels.Height) - 1
	if err := d.drawCorner(0, 0, lineSize, lineSize); err != nil {
		return err
	}
	if err := d.drawCorner(0, lineY, lineSize, -lineSize); err != nil {
		return err
	}
	if err := d.drawCorner(lineX, 0, -lineSize, lineSize); err != nil {
		return err
	}
	if err := d.drawCorner(lineX, lineY, -lineSize, -lineSize); err != nil {
		return err
	}
	return osd.TransactionCommit()
}

func (d *settingsDialog) applyTheme() {
	r, g, b, _ := theme.BackgroundColor().RGBA()
	bg := &color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 230}
	d.bg.FillColor = bg
}

func (d *settingsDialog) onChanged() {
	d.changed = true
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
		onClosed(d.changed)
	}
}
