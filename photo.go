package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/disintegration/imaging"
)

// File date choices
const (
	ChoiceExifCreateDate = iota
	ChoiceFileModifyDate
	ChoiceManualInputDate
)
const DateFormat = "2006:01:02 03:04:05"

// Photo
type Photo struct {
	File       string
	Drop       bool
	Img        *canvas.Image
	Dates      [3]string
	DateChoice int
}

// frame column that contains button with photo image as background and date fix input
func (p *Photo) FrameColumn() *fyne.Container {
	fileLabel := widget.NewLabelWithStyle(filepath.Base(p.File), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	column := container.NewBorder(fileLabel, p.dateInput(), nil, nil, p.imgButton())
	return column
}

// button with photo image as background
func (p *Photo) imgButton() *fyne.Container {
	var btn *widget.Button
	btn = widget.NewButton(
		"",
		func() {
			if p.Drop {
				btn.SetText("")
				p.Img.Translucency = 0
				p.Drop = false
			} else {
				btn.SetText("DROPPED")
				p.Img.Translucency = 0.5
				p.Drop = true
			}
		},
	)
	if p.Drop {
		btn.SetText("DROPPED")
		p.Img.Translucency = 0.5
	}
	return container.NewMax(p.Img, btn)
}

// single photo date fix input
func (p *Photo) dateInput() *fyne.Container {
	d := p.Dates[p.DateChoice]

	eDate := widget.NewEntry()
	eDate.SetText(d)
	eDate.Disable()

	rgDateChoice := widget.NewRadioGroup(
		[]string{"EXIF", "File", "Input"},
		func(s string) {
			switch s {
			case "EXIF":
				p.Dates[ChoiceManualInputDate] = ""
				p.DateChoice = ChoiceExifCreateDate
				eDate.SetText(p.Dates[p.DateChoice])
				eDate.Disable()
			case "File":
				p.Dates[ChoiceManualInputDate] = ""
				p.DateChoice = ChoiceFileModifyDate
				eDate.SetText(p.Dates[p.DateChoice])
				eDate.Disable()
			case "Input":
				p.DateChoice = ChoiceManualInputDate
				if p.Dates[p.DateChoice] == "" {
					p.Dates[p.DateChoice] = p.Dates[ChoiceExifCreateDate]
				}
				eDate.SetText(p.Dates[p.DateChoice])
				eDate.Enable()
			}
		})
	switch p.DateChoice {
	case ChoiceExifCreateDate:
		rgDateChoice.SetSelected("EXIF")
	case ChoiceFileModifyDate:
		rgDateChoice.SetSelected("File")
	case ChoiceManualInputDate:
		rgDateChoice.SetSelected("Input")
	}
	rgDateChoice.Horizontal = true

	gr := container.NewVBox(rgDateChoice, eDate)

	return container.NewCenter(gr)
}

// get canvas image from file
func (p *Photo) img(scale int) (img *canvas.Image) {
	m, err := imaging.Open(p.File, imaging.AutoOrientation(true))
	if err != nil {
		log.Fatal(err)
	}
	if scale > 1 {
		width := (m.Bounds().Max.X - m.Bounds().Min.X) / scale
		m = imaging.Resize(m, width, 0, imaging.Lanczos)
	}
	img = canvas.NewImageFromImage(m)
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScalePixels
	// img.ScaleMode = canvas.ImageScaleFastest
	return
}

// get file modify date string
func (p *Photo) getModifyDate() string {
	fi, err := os.Stat(p.File)
	if err != nil {
		return ""
	}
	fileModifyDate := fi.ModTime()
	return fileModifyDate.Format(DateFormat)
}
