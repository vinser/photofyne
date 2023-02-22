package main

import (
	"encoding/json"
	"image/color"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	systemThemeName = "system default"
)

// Settings gives access to user interfaces to control Fyne settings
type Settings struct {
	fyneSettings app.SettingsSchema
	colors       []fyne.CanvasObject
	userTheme    fyne.Theme
}

func (s *Settings) load() {
	err := s.loadFromFile(s.fyneSettings.StoragePath())
	if err != nil {
		fyne.LogError("Settings load error:", err)
	}
}

func (s *Settings) loadFromFile(path string) error {
	file, err := os.Open(path) // #nosec
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(filepath.Dir(path), 0700)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	decode := json.NewDecoder(file)

	return decode.Decode(&s.fyneSettings)
}

func (s *Settings) save() error {
	return s.saveToFile(s.fyneSettings.StoragePath())
}

func (s *Settings) saveToFile(path string) error {
	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil { // this is not an exists error according to docs
		return err
	}

	data, err := json.Marshal(&s.fyneSettings)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}

func settingsScreen() {
	w := fyne.CurrentApp().NewWindow("Fyne Settings")
	w.SetContent(newSettings().LoadAppearanceScreen(w))
	w.Resize(fyne.NewSize(480, 480))
	w.Show()
}

// NewSettings returns a new settings instance with the current configuration loaded
func newSettings() *Settings {
	s := &Settings{}
	s.load()
	if s.fyneSettings.Scale == 0 {
		s.fyneSettings.Scale = 1
	}
	return s
}

// LoadAppearanceScreen creates a new settings screen to handle appearance configuration
func (s *Settings) LoadAppearanceScreen(w fyne.Window) fyne.CanvasObject {
	s.userTheme = fyne.CurrentApp().Settings().Theme()
	if s.userTheme == nil {
		s.userTheme = theme.DefaultTheme()
	}

	def := s.fyneSettings.ThemeName
	themeNames := []string{"dark", "light"}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		themeNames = append(themeNames, systemThemeName)
		if s.fyneSettings.ThemeName == "" {
			def = systemThemeName
		}
	}
	themes := widget.NewSelect(themeNames, s.chooseTheme)
	themes.SetSelected(def)

	scale := s.makeScaleGroup(w.Canvas().Scale())
	box := container.NewVBox(scale)

	for _, c := range theme.PrimaryColorNames() {
		b := newColorButton(c, theme.PrimaryColorNamed(c), s)
		s.colors = append(s.colors, b)
	}
	swatch := container.NewGridWithColumns(len(s.colors), s.colors...)
	appearance := widget.NewForm(widget.NewFormItem("Main Color", swatch),
		widget.NewFormItem("Theme", themes))

	box.Add(widget.NewCard("Appearance", "", appearance))
	bottom := container.NewHBox(layout.NewSpacer(),
		&widget.Button{Text: "Apply", Importance: widget.HighImportance, OnTapped: s.applySettings})

	// return container.NewBorder(box, bottom, nil, nil, s.preview)
	return container.NewVBox(box, bottom)
}

func (s *Settings) applySettings() {
	if s.fyneSettings.Scale == 0.0 {
		s.chooseScale(1.0)
	}
	err := s.save()
	if err != nil {
		fyne.LogError("Failed on saving", err)
	}

	s.appliedScale(s.fyneSettings.Scale)
}

func (s *Settings) chooseTheme(name string) {
	if name == systemThemeName {
		name = ""
	}
	s.fyneSettings.ThemeName = name
}

type colorButton struct {
	widget.BaseWidget
	name  string
	color color.Color

	s *Settings
}

func newColorButton(n string, c color.Color, s *Settings) *colorButton {
	b := &colorButton{name: n, color: c, s: s}
	b.ExtendBaseWidget(b)
	return b
}

func (c *colorButton) CreateRenderer() fyne.WidgetRenderer {
	r := canvas.NewRectangle(c.color)
	r.StrokeWidth = 2

	if c.name == c.s.fyneSettings.PrimaryColor {
		r.StrokeColor = theme.PrimaryColor()
	}

	return &colorRenderer{c: c, rect: r, objs: []fyne.CanvasObject{r}}
}

func (c *colorButton) Tapped(_ *fyne.PointEvent) {
	c.s.fyneSettings.PrimaryColor = c.name
	for _, child := range c.s.colors {
		child.Refresh()
	}
}

type colorRenderer struct {
	c    *colorButton
	rect *canvas.Rectangle
	objs []fyne.CanvasObject
}

func (c *colorRenderer) Layout(s fyne.Size) {
	c.rect.Resize(s)
}

func (c *colorRenderer) MinSize() fyne.Size {
	return fyne.NewSize(20, 20)
}

func (c *colorRenderer) Refresh() {
	if c.c.name == c.c.s.fyneSettings.PrimaryColor {
		c.rect.StrokeColor = theme.PrimaryColor()
	} else {
		c.rect.StrokeColor = color.Transparent
	}
	c.rect.FillColor = c.c.color

	c.rect.Refresh()
}

func (c *colorRenderer) Objects() []fyne.CanvasObject {
	return c.objs
}

func (c *colorRenderer) Destroy() {
}

type scaleItems struct {
	scale   float32
	name    string
	preview *canvas.Text
	button  *widget.Button
}

var scales = []*scaleItems{
	{scale: 0.5, name: "Tiny"},
	{scale: 0.8, name: "Small"},
	{scale: 1, name: "Normal"},
	{scale: 1.3, name: "Large"},
	{scale: 1.8, name: "Huge"}}

func (s *Settings) appliedScale(value float32) {
	for _, scale := range scales {
		scale.preview.TextSize = theme.TextSize() * scale.scale / value
	}
}

func (s *Settings) chooseScale(value float32) {
	s.fyneSettings.Scale = value

	for _, scale := range scales {
		if scale.scale == value {
			scale.button.Importance = widget.HighImportance
		} else {
			scale.button.Importance = widget.MediumImportance
		}

		scale.button.Refresh()
	}
}

func (s *Settings) makeScaleButtons() []fyne.CanvasObject {
	var buttons = make([]fyne.CanvasObject, len(scales))
	for i, scale := range scales {
		value := scale.scale
		button := widget.NewButton(scale.name, func() {
			s.chooseScale(value)
		})
		if s.fyneSettings.Scale == scale.scale {
			button.Importance = widget.HighImportance
		}

		scale.button = button
		buttons[i] = button
	}

	return buttons
}

func (s *Settings) makeScaleGroup(scale float32) *widget.Card {
	scalePreviewBox := container.NewGridWithColumns(5, s.makeScalePreviews(scale)...)
	scaleBox := container.NewGridWithColumns(5, s.makeScaleButtons()...)

	return widget.NewCard("Scale", "", container.NewVBox(scalePreviewBox, scaleBox, newRefreshMonitor(s)))
}

func (s *Settings) makeScalePreviews(value float32) []fyne.CanvasObject {
	var previews = make([]fyne.CanvasObject, len(scales))
	for i, scale := range scales {
		text := canvas.NewText("A", theme.ForegroundColor())
		text.Alignment = fyne.TextAlignCenter
		text.TextSize = theme.TextSize() * scale.scale / value

		scale.preview = text
		previews[i] = text
	}

	return previews
}

func (s *Settings) refreshScalePreviews() {
	for _, scale := range scales {
		scale.preview.Color = theme.ForegroundColor()
	}
}

// refreshMonitor is a simple widget that updates canvas components when the UI is asked to refresh.
// Captures theme and scale changes without the settings monitoring code.
type refreshMonitor struct {
	widget.Label
	settings *Settings
}

func (r *refreshMonitor) Refresh() {
	r.settings.refreshScalePreviews()
	r.Label.Refresh()
}

func newRefreshMonitor(s *Settings) *refreshMonitor {
	r := &refreshMonitor{settings: s}
	r.Hide()
	return r
}
