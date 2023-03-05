package main

import (
	"os"
	"path/filepath"
	"strings"

	"image/color"
	_ "image/jpeg"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
)

var version, buildTime, target, goversion string

var wMain fyne.Window

var pl *PhotoList

func main() {
	a := app.NewWithID("com.github/vinser/photofine")
	t := &Theme{}

	a.Settings().SetTheme(t)

	wMain = a.NewWindow(strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0])))

	wd, _ := os.Getwd()
	pl = newPhotoList(a.Preferences().StringWithFallback("folder", wd))
	MainLayout(pl)
	wMain.Resize(fyne.NewSize(1344, 756))
	wMain.CenterOnScreen()
	wMain.SetMaster()
	wMain.Show()
	a.Run()
}

// Application custom theme and interface inplementation
type Theme struct{}

func (t *Theme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameButton:
		return color.Transparent
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *Theme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Monospace {
		return regular
	}
	if style.Bold {
		if style.Italic {
			return bolditalic
		}
		return bold
	}
	if style.Italic {
		return italic
	}
	if style.Symbol {
		return regular
	}
	return regular
}

func (t *Theme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *Theme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
