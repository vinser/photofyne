package main

import (
	"encoding/json"
	"image/color"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Settings gives access to user interfaces to control Fyne settings
type Settings struct {
	fyneSettings app.SettingsSchema
	colors       []fyne.CanvasObject
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

	return os.WriteFile(path, data, 0644)
}

func settingsScreen() {
	w := fyne.CurrentApp().NewWindow("Appearance")
	w.SetContent(newSettings().LoadAppearanceScreen(w))
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
	scalePreviewRow := container.NewGridWithColumns(len(scales), s.makeScalePreviews(w.Canvas().Scale())...)
	scaleRow := container.NewGridWithColumns(len(scales), s.makeScaleButtons()...)
	for _, c := range theme.PrimaryColorNames() {
		b := newColorButton(c, theme.PrimaryColorNamed(c), s)
		s.colors = append(s.colors, b)
	}
	colorRow := container.NewGridWithColumns(len(s.colors), s.colors...)
	// themeRow := s.makeThemes()
	themeRow := s.makeThemes()
	appearance := widget.NewForm(
		widget.NewFormItem("", scalePreviewRow),
		widget.NewFormItem("Scale", scaleRow),
		widget.NewFormItem("Main Color", colorRow),
		widget.NewFormItem("Theme", themeRow),
	)

	apply := container.NewHBox(layout.NewSpacer(),
		&widget.Button{Text: "Apply", Importance: widget.HighImportance, OnTapped: s.applySettings})

	return container.NewVBox(appearance, apply)
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
