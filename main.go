package main

import (
	"os"

	"image/color"
	_ "image/jpeg"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
)

var wMain fyne.Window

func main() {
	a := app.NewWithID("com.github/vinser/photofine")
	t := &AppTheme{}

	a.Settings().SetTheme(t)

	wMain = a.NewWindow("Photos")
	wMain.Resize(fyne.NewSize(1344, 756))
	wMain.CenterOnScreen()

	wd, _ := os.Getwd()
	MainLayout(a.Preferences().StringWithFallback("folder", wd))
	wMain.Show()
	a.Run()
}

// open photo folder dialog
func ChooseFolder() {
	folder := ""

	fd := dialog.NewFolderOpen(func(list fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, wMain)
			return
		}
		if list == nil {
			wMain.Close()
			return
		}
		folder = list.Path()
		fyne.CurrentApp().Preferences().SetString("folder", folder)
		MainLayout(folder)
	}, wMain)
	wd, _ := os.Getwd()
	savedLocation := fyne.CurrentApp().Preferences().StringWithFallback("folder", wd)
	locationUri, _ := storage.ListerForURI(storage.NewFileURI(savedLocation))
	fd.SetLocation(locationUri)
	fd.Resize(fyne.NewSize(672, 378))
	fd.Show()
}

// make main window layout
func MainLayout(folder string) {
	pl := NewPhotoList(folder)

	contentTabs := container.NewAppTabs(pl.NewChoiceTab(), pl.NewListTab())
	contentTabs.SetTabLocation(container.TabLocationBottom)

	wMain.SetContent(container.NewBorder(nil, nil, nil, nil, contentTabs))
}

// Application custom theme and interface inplementation
type AppTheme struct{}

func (t AppTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	// case theme.ColorNameInputBackground:
	// 	return color.Transparent
	case theme.ColorNameButton:
		return color.Transparent
	}
	a := fyne.CurrentApp()
	switch a.Preferences().StringWithFallback("theme", "LightTheme") {
	case "LightTheme":
		variant = theme.VariantLight
	case "DarkTheme":
		variant = theme.VariantDark
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

// switch theme light-dark
func SwitchTheme() {
	p := fyne.CurrentApp().Preferences()
	switch p.StringWithFallback("theme", "LightTheme") {
	case "LightTheme":
		p.SetString("theme", "DarkTheme")
	case "DarkTheme":
		p.SetString("theme", "LightTheme")
	}
}
