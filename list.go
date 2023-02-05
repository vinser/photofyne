package main

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

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

// create new PhotoList object for the folder
func NewPhotoList(folder string) PhotoList {
	folder, _ = filepath.Abs(folder)
	files, err := os.ReadDir(folder)
	if err != nil {
		log.Fatalf("Can't list photo files from folder \"%s\". Error: %v\n", folder, err)
	}
	photos := []*Photo(nil)
	for _, f := range files {
		if isPhotoFile(f.Name()) {
			photo := &Photo{
				File:       filepath.Join(folder, f.Name()),
				Drop:       false,
				DateChoice: ChoiceExifCreateDate,
				Dates:      [3]string{},
			}
			photo.Dates[ChoiceExifCreateDate] = getExifDate(photo.File)
			photo.Dates[ChoiceFileModifyDate] = photo.getModifyDate()
			if len(photo.Dates[ChoiceExifCreateDate]) != len(DateFormat) {
				photo.DateChoice = ChoiceFileModifyDate
			}

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
func (l *PhotoList) SaveList() {
	dialog.ShowConfirm("Ready to save changes", "Proceed?",
		func(b bool) {
			if b {
				dropDirOk := false
				dropDirName := filepath.Join(l.Folder, "dropped")
				backupDirOk := false
				backupDirName := filepath.Join(l.Folder, "original")
				for _, p := range l.List {
					if p.Drop {
						// move file to drop dir
						if !dropDirOk {
							err := os.Mkdir(dropDirName, 0775)
							if err != nil && !errors.Is(err, fs.ErrExist) {
								dialog.ShowError(err, wMain)
							}
						}
						os.Rename(p.File, filepath.Join(dropDirName, filepath.Base(p.File)))
						continue
					}
					if p.DateChoice != ChoiceExifCreateDate {
						// backup original file and make file copy with modified exif
						if !backupDirOk {
							err := os.Mkdir(backupDirName, 0775)
							if err != nil && !errors.Is(err, fs.ErrExist) {
								dialog.ShowError(err, wMain)
							}
						}
						updateExifDate(p.File, backupDirName, p.Dates[p.DateChoice])
					}
				}
			}
		},
		wMain)
}

// create new photos tab container
func (l *PhotoList) NewListTab() *container.TabItem {
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.FolderOpenIcon(), ChooseFolder),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), l.SaveList),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.VisibilityIcon(), SwitchTheme),
		// widget.NewToolbarAction(theme.HelpIcon(), func() {
		// 	log.Println("Display help")
		// }),
	)
	listTitle := []string{"File Name", "Exif Date", "File Date", "Entry Date", "Dropped"}
	list := widget.NewTable(
		func() (int, int) {
			return len(l.List) + 1, len(listTitle)
		},
		func() fyne.CanvasObject {
			text := DateFormat
			for _, ph := range l.List {
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
				ph := l.List[i.Row-1]
				switch i.Col {
				case 0:
					text = filepath.Base(ph.File)
					w.TextStyle.Bold = false
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
func (l *PhotoList) NewChoiceTab() *container.TabItem {
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentRemoveIcon(), func() {
			l.resizeFrame(RemoveColumn)
		}),
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			l.resizeFrame(AddColumn)
		}),
		// widget.NewToolbarSpacer(),
		// widget.NewToolbarAction(theme.HelpIcon(), func() {
		// 	log.Println("Display help")
		// }),
	)
	toolBar.Items[1].ToolbarObject().Hide()

	prevPhotoBtn := widget.NewButton("<", func() {
		l.scrollFrame(l.FramePos - 1)
	})
	prevFrameBtn := widget.NewButton("<<", func() {
		l.scrollFrame(l.FramePos - l.FrameSize)
	})
	firstPhotoBtn := widget.NewButton("|<", func() {
		l.scrollFrame(0)
	})

	nextPhotoBtn := widget.NewButton(">", func() {
		l.scrollFrame(l.FramePos + 1)
	})
	nextFrameBtn := widget.NewButton(">>", func() {
		l.scrollFrame(l.FramePos + l.FrameSize)
	})
	lastPhotoBtn := widget.NewButton(">|", func() {
		l.scrollFrame(len(l.List))
	})
	bottomButtons := container.NewGridWithColumns(6, firstPhotoBtn, prevFrameBtn, prevPhotoBtn, nextPhotoBtn, nextFrameBtn, lastPhotoBtn)

	return container.NewTabItemWithIcon("Choice", theme.GridIcon(), container.NewBorder(toolBar, bottomButtons, nil, nil, l.Frame))
}

// scroll frame at position pos
func (l *PhotoList) scrollFrame(pos int) {
	s := *l

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
	for i := 0; i < l.FrameSize; i++ {
		s.Frame.Add(s.List[pos+i].FrameColumn())
	}
	s.Frame.Refresh()

	s.FramePos = pos
	*l = s
}

// resize frame
func (l *PhotoList) resizeFrame(zoom int) {
	s := *l

	switch zoom {
	case RemoveColumn:
		if s.FrameSize-1 < MinFrameSize {
			return
		}
		s.List[s.FramePos+s.FrameSize-1].Img = nil
		s.FrameSize--
	case AddColumn:
		if s.FrameSize+1 > MaxFrameSize || s.FrameSize+1 > len(s.List) {
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
	*l = s
}

// fill frame Num photo images starting with Pos = 0.
func (l *PhotoList) initFrame() {
	s := *l
	s.FramePos = 0
	if s.FrameSize > len(s.List) {
		s.FrameSize = len(s.List)
	}
	for i := s.FramePos; i < s.FramePos+s.FrameSize && i < len(s.List); i++ {
		s.List[i].Img = s.List[i].img(s.FrameSize)
	}
	*l = s
}

// detect whether file name is jpeg photo image
func isPhotoFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".jpg") || strings.HasSuffix(strings.ToLower(filename), ".jpeg")
}
