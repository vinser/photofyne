package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"image/color"
	_ "image/jpeg"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/disintegration/imaging"
	exif "github.com/dsoprea/go-jpeg-image-structure/v2"
)

var wMain fyne.Window

func main() {
	a := app.New()
	a.Settings().SetTheme(&AppTheme{})

	wMain = a.NewWindow("Photos")
	wMain.Resize(fyne.NewSize(1344, 756))
	wMain.CenterOnScreen()

	if len(os.Args) > 1 {
		MainLayout(wMain, os.Args[1])
	} else {
		ChooseFolder(wMain)
	}
	wMain.Show()
	a.Run()
}

// open photo folder dialog
func ChooseFolder(w fyne.Window) {
	folder := ""

	fd := dialog.NewFolderOpen(func(list fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if list == nil {
			w.Close()
			return
		}

		// children, err := list.List()
		// if err != nil {
		// 	dialog.ShowError(err, w)
		// 	w.Close()
		// 	return
		// }
		// out := fmt.Sprintf("Folder %s (%d children):\n%s", list.Name(), len(children), list.String())
		// dialog.ShowInformation("Folder Open", out, w)

		folder = list.Path()
		MainLayout(w, folder)
	}, w)
	fd.Resize(fyne.NewSize(672, 378))
	fd.Show()
}

// make main window layout
func MainLayout(w fyne.Window, folder string) {
	pl := NewPhotoList(folder)

	contentTabs := container.NewAppTabs(pl.NewListTab(), pl.NewChoiceTab())
	contentTabs.SetTabLocation(container.TabLocationBottom)

	w.SetContent(container.NewBorder(nil, nil, nil, nil, contentTabs))
}

// Application custom theme
type AppTheme struct{}

func (t AppTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameInputBackground {
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

const (
	InitListPos   = 0
	InitFrameSize = 3
	MinFrameSize  = 1
	MaxFrameSize  = 6
)

const (
	AddColumn = iota
	RemoveColumn
)

// Photolist
type PhotoList struct {
	Folder    string
	List      []*Photo
	Frame     *fyne.Container
	FrameSize int
	FramePos  int
}

// create new PhotoList object for folder
func NewPhotoList(folder string) PhotoList {
	folder, _ = filepath.Abs(folder)
	files, err := os.ReadDir(folder)
	if err != nil {
		log.Fatalf("Can't list photo files from folder \"%s\". Error: %v\n", folder, err)
	}
	photos := []*Photo(nil)
	// et, err := exiftool.NewExiftool()
	// if err != nil {
	// 	// TODO: work without exiftool installed. Only file modify date
	// 	log.Fatalf("Error when intializing: %v\n", err)
	// }
	// defer et.Close()

	for _, f := range files {
		if isPhotoFile(f.Name()) {
			photo := &Photo{
				File:       filepath.Join(folder, f.Name()),
				Drop:       false,
				DateChoice: NoChoice,
				Dates:      [3]string{},
			}
			// photo.Dates[ChoiceExifCreateDate] = photo.exifDate(et)
			photo.Dates[ChoiceExifCreateDate] = photo.exifDate()
			photo.Dates[ChoiceFileModifyDate] = photo.modifyDate()

			photos = append(photos, photo)
		}
	}
	colrows := InitFrameSize
	if InitFrameSize > len(photos) {
		colrows = len(photos)
	}
	photoList := PhotoList{
		Folder:    folder,
		List:      photos,
		Frame:     container.NewAdaptiveGrid(colrows),
		FrameSize: InitFrameSize,
		FramePos:  InitListPos,
	}
	photoList.initFrame()

	for i := 0; i < photoList.FrameSize && i < len(photoList.List); i++ {
		photoList.Frame.Add(photoList.List[InitListPos+i].FrameColumn())
		// photo := photoList.List[InitPos+i]
		// vBox := container.New(layout.NewMaxLayout(), imgButton(photo), dateCheck(photo))
		// photoList.Frame.Add(vBox)
	}

	return photoList
}

// Save choosed photos:
// 1. move dropped photo to droppped folder
// 2. update exif dates with file modify date or input date
func (p *PhotoList) SaveList() {
	dialog.ShowConfirm("Ready to save changes", "Proceed?", func(b bool) {
		if b {
			if p.hasDropped() {
				err := os.Mkdir(filepath.Join(p.Folder, "dropped"), 0775)
				if err != nil {
					dialog.ShowError(err, wMain)
					return
				}
				for _, v := range p.List {
					if v.Drop {
						os.Rename(v.File, filepath.Join(filepath.Dir(v.File), "dropped", filepath.Base(v.File)))
					}
					if v.DateChoice != NoChoice && v.DateChoice != ChoiceExifCreateDate {
						v.updateExif()
					}
				}
			}
		}
	}, wMain)
}

// Check if list has photos marked to drop
func (p *PhotoList) hasDropped() bool {
	for _, v := range p.List {
		if v.Drop {
			return true
		}
	}
	return false
}

// create new photos tab container
func (p *PhotoList) NewListTab() *container.TabItem {
	toolBar := widget.NewToolbar(
		// widget.NewToolbarAction(theme.FolderOpenIcon(), p.ChooseDir),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), p.SaveList),
		// widget.NewToolbarSpacer(),
		// widget.NewToolbarAction(theme.HelpIcon(), func() {
		// 	log.Println("Display help")
		// }),
	)
	listTitle := []string{"File Name", "Exif Date", "File Date", "Entry Date", "Dropped"}
	list := widget.NewTable(
		func() (int, int) {
			return len(p.List) + 1, len(listTitle)
		},
		func() fyne.CanvasObject {
			text := DateFormat
			for _, ph := range p.List {
				fName := filepath.Base(ph.File)
				if len(fName) > len(text) {
					text = fName
				}
			}
			return widget.NewLabel(text)
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			w := o.(*widget.Label)
			w.Alignment = fyne.TextAlignCenter
			text := ""
			if i.Row == 0 {
				text = listTitle[i.Col]
				w.TextStyle.Bold = true
			} else {
				ph := p.List[i.Row-1]
				switch i.Col {
				case 0:
					text = filepath.Base(ph.File)
				case 1, 2, 3:
					text = ph.Dates[i.Col-1]
					if i.Col-1 == ph.DateChoice {
						w.TextStyle.Bold = true
					} else {
						w.TextStyle.Bold = false
					}
				case 4:
					if ph.Drop {
						text = "Yes"
						w.TextStyle.Bold = true
					}
				}
			}
			w.SetText(text)
		})

	return container.NewTabItemWithIcon("List", theme.ListIcon(), container.NewBorder(toolBar, nil, nil, nil, list))
}

// create new photos tab container
func (p *PhotoList) NewChoiceTab() *container.TabItem {
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentRemoveIcon(), func() {
			p.resizeFrame(RemoveColumn)
		}),
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			p.resizeFrame(AddColumn)
		}),
		// widget.NewToolbarSpacer(),
		// widget.NewToolbarAction(theme.HelpIcon(), func() {
		// 	log.Println("Display help")
		// }),
	)
	toolBar.Items[1].ToolbarObject().Hide()

	prevPhotoBtn := widget.NewButton("<", func() {
		p.scrollFrame(p.FramePos - 1)
	})
	prevFrameBtn := widget.NewButton("<<", func() {
		p.scrollFrame(p.FramePos - p.FrameSize)
	})
	firstPhotoBtn := widget.NewButton("|<", func() {
		p.scrollFrame(0)
	})

	nextPhotoBtn := widget.NewButton(">", func() {
		p.scrollFrame(p.FramePos + 1)
	})
	nextFrameBtn := widget.NewButton(">>", func() {
		p.scrollFrame(p.FramePos + p.FrameSize)
	})
	lastPhotoBtn := widget.NewButton(">|", func() {
		p.scrollFrame(len(p.List))
	})
	bottomButtons := container.NewGridWithColumns(6, firstPhotoBtn, prevFrameBtn, prevPhotoBtn, nextPhotoBtn, nextFrameBtn, lastPhotoBtn)

	return container.NewTabItemWithIcon("Choice", theme.GridIcon(), container.NewBorder(toolBar, bottomButtons, nil, nil, p.Frame))
}

// scroll frame at position pos
func (p *PhotoList) scrollFrame(pos int) {
	s := *p

	switch {
	case pos < 0:
		pos = 0
	case pos > len(s.List)-s.FrameSize:
		pos = len(s.List) - s.FrameSize
	}

	switch {
	case pos-s.FramePos >= s.FrameSize || s.FramePos-pos >= s.FrameSize:
		for i := s.FramePos; i < s.FramePos+s.FrameSize; i++ {
			s.List[i].Img = nil
		}
		for i := pos; i < pos+s.FrameSize; i++ {
			s.List[i].Img = s.List[i].img(s.FrameSize)
			if s.List[i].Drop {
				s.List[i].Img.Translucency = 0.5
			}
		}
	case pos > s.FramePos:
		for i := s.FramePos; i < pos; i++ {
			s.List[i].Img = nil
			s.List[i+s.FrameSize].Img = s.List[i+s.FrameSize].img(s.FrameSize)
			if s.List[i+s.FrameSize].Drop {
				s.List[i+s.FrameSize].Img.Translucency = 0.5
			}
		}
	case s.FramePos > pos:
		for i := pos; i < s.FramePos; i++ {
			s.List[i+s.FrameSize].Img = nil
			s.List[i].Img = s.List[i].img(s.FrameSize)
			if s.List[i].Drop {
				s.List[i].Img.Translucency = 0.5
			}
		}
	}

	// TODO: may be optimized when for scroll les than frame size by not all objects deletion/addition? Somwthing like this:
	// https://stackoverflow.com/questions/63995289/how-to-remove-objects-from-golang-fyne-container
	s.Frame.RemoveAll()
	for i := 0; i < p.FrameSize; i++ {
		s.Frame.Add(s.List[pos+i].FrameColumn())
	}
	s.Frame.Refresh()

	s.FramePos = pos
	*p = s
}

// resize frame
func (p *PhotoList) resizeFrame(zoom int) {
	s := *p

	switch zoom {
	case RemoveColumn:
		if s.FrameSize-1 < MinFrameSize {
			return
		}
		s.List[s.FramePos+s.FrameSize-1].Img = nil
		s.FrameSize--
	case AddColumn:
		if s.FrameSize+1 > MaxFrameSize {
			return
		}
		i := s.FramePos + s.FrameSize
		if i == len(s.List) {
			s.FramePos--
			i = s.FramePos
		}
		s.List[i].Img = s.List[i].img(s.FrameSize)
		if s.List[i].Drop {
			s.List[i].Img.Translucency = 0.5
		}
		s.FrameSize++
	}
	//      0-1-2-3-4-5-6-7-8
	//          2-3-4			p=2, s=3
	// 		0-1-2				p=0, s=3
	// 					6-7-8	p=6, s=3

	// TODO: may be optimized when for scroll les than frame size by not all objects deletion/addition? Somwthing like this:
	// https://stackoverflow.com/questions/63995289/how-to-remove-objects-from-golang-fyne-container
	s.Frame.RemoveAll()
	for i := 0; i < s.FrameSize; i++ {
		s.Frame.Add(s.List[s.FramePos+i].FrameColumn())
	}
	s.Frame.Layout = layout.NewAdaptiveGridLayout(len(s.Frame.Objects))
	s.Frame.Refresh()

	// s.FrameSize = zoom
	*p = s
}

// fill frame Num photo images starting with Pos = 0.
func (p *PhotoList) initFrame() {
	s := *p
	s.FramePos = 0
	if s.FrameSize > len(s.List) {
		s.FrameSize = len(s.List)
	}
	for i := s.FramePos; i < s.FramePos+s.FrameSize && i < len(s.List); i++ {
		s.List[i].Img = s.List[i].img(s.FrameSize)
	}
	*p = s
}

// detect whether file name is jpeg photo image
func isPhotoFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".jpg")
}

// File date choices
const (
	NoChoice = iota - 1
	ChoiceExifCreateDate
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
	if p.DateChoice == NoChoice {
		p.DateChoice = ChoiceExifCreateDate
	}
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
func (p *Photo) modifyDate() string {
	fi, err := os.Stat(p.File)
	if err != nil {
		return ""
	}
	fileModifyDate := fi.ModTime()
	return fileModifyDate.Format(DateFormat)
}

// update EXIF and file dates
func (p *Photo) exifDate() string {
	// Parse the image.

	jmp := exif.NewJpegMediaParser()

	intfc, err := jmp.ParseFile(p.File)
	if err != nil {
		return ""
	}

	sl := intfc.(*exif.SegmentList)
	_, _, exifTags, err := sl.DumpExif()
	if err != nil {
		return ""
	}

	for _, et := range exifTags {
		if et.IfdPath == "IFD/Exif" && (et.TagName == "DateTimeOriginal" || et.TagName == "DateTimeDigitized") {
			return et.FormattedFirst
		}
	}
	return ""
}

// update EXIF dates
func (p *Photo) updateExif() {
}
