package main

import (
	"os"

	"image/color"
	_ "image/jpeg"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

var version, buildTime, target, goversion string

var wMain fyne.Window

var pl *PhotoList

func main() {
	a := app.NewWithID("com.github/vinser/photofine")
	t := &AppTheme{}

	a.Settings().SetTheme(t)

	wMain = a.NewWindow("Photos")
	wMain.Resize(fyne.NewSize(1344, 756))
	wMain.CenterOnScreen()

	wd, _ := os.Getwd()
	pl = newPhotoList(a.Preferences().StringWithFallback("folder", wd))
	MainLayout(pl)
	wMain.SetMaster()
	wMain.Show()
	a.Run()
}

// make main window layout
func MainLayout(pl *PhotoList) {

	contentTabs := container.NewAppTabs(pl.newChoiceTab(), pl.newListTab())
	contentTabs.SetTabLocation(container.TabLocationBottom)

	wMain.SetContent(container.NewBorder(nil, nil, nil, nil, contentTabs))
}

// Application custom theme and interface inplementation
type AppTheme struct{}

func (t AppTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameButton:
		return color.Transparent
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t AppTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t AppTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t AppTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
