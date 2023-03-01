package main

import (
	"errors"
	"image/color"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
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
func newPhotoList(folder string) *PhotoList {
	folder, _ = filepath.Abs(folder)
	files, err := os.ReadDir(folder)
	if err != nil {
		log.Fatalf("Can't list photo files from folder \"%s\". Error: %v\n", folder, err)
	}
	photos := []*Photo(nil)
	for _, f := range files {
		fName := strings.ToLower(f.Name())
		if strings.HasSuffix(fName, ".jpg") || strings.HasSuffix(fName, ".jpeg") {
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
	columns := InitFrameSize
	if InitFrameSize > len(photos) {
		columns = len(photos)
	}
	photoList := PhotoList{
		Folder:    folder,
		List:      photos,
		Frame:     container.NewGridWithColumns(columns),
		FrameSize: InitFrameSize,
		FramePos:  InitListPos,
	}
	photoList.initFrame()

	if columns == 0 { // Workaround for NewGridWithColumns(0) main window shrink on Windows OS
		photoList.Frame = container.NewGridWithColumns(1, canvas.NewText("", color.Black))
	}
	for i := 0; i < photoList.FrameSize && i < len(photoList.List); i++ {
		photoList.Frame.Add(photoList.List[InitListPos+i].FrameColumn())
	}

	return &photoList
}

// make main window layout
func MainLayout(pl *PhotoList) {

	contentTabs := container.NewAppTabs(pl.newChoiceTab(), pl.newListTab())
	contentTabs.SetTabLocation(container.TabLocationBottom)

	// wMain.SetContent(container.NewBorder(nil, nil, nil, nil, contentTabs))
	wMain.SetContent(contentTabs)
}

// Save choosed photos:
// 1. move dropped photo to droppped folder
// 2. update exif dates with file modify date or input date
func (l *PhotoList) savePhotoList() {
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
func (l *PhotoList) newListTab() *container.TabItem {
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.FolderOpenIcon(), chooseFolder),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), l.savePhotoList),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.SettingsIcon(), settingsScreen),
		widget.NewToolbarAction(theme.HelpIcon(), aboutScreen),
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
func (l *PhotoList) newChoiceTab() *container.TabItem {
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

	switch {
	case pos < 0:
		pos = 0
	case pos > len(l.List)-l.FrameSize:
		pos = len(l.List) - l.FrameSize
	}

	switch {
	case pos-l.FramePos >= l.FrameSize || l.FramePos-pos >= l.FrameSize:
		for i := l.FramePos; i < l.FramePos+l.FrameSize; i++ {
			l.List[i].Img = nil
		}
		for i := pos; i < pos+l.FrameSize; i++ {
			l.List[i].Img = l.List[i].img(l.FrameSize)
			if l.List[i].Drop {
				l.List[i].Img.Translucency = 0.5
			}
		}
	case pos > l.FramePos:
		for i := l.FramePos; i < pos; i++ {
			l.List[i].Img = nil
			l.List[i+l.FrameSize].Img = l.List[i+l.FrameSize].img(l.FrameSize)
			if l.List[i+l.FrameSize].Drop {
				l.List[i+l.FrameSize].Img.Translucency = 0.5
			}
		}
	case l.FramePos > pos:
		for i := pos; i < l.FramePos; i++ {
			l.List[i+l.FrameSize].Img = nil
			l.List[i].Img = l.List[i].img(l.FrameSize)
			if l.List[i].Drop {
				l.List[i].Img.Translucency = 0.5
			}
		}
	}

	// TODO: may be optimized when for scroll les than frame size by not all objects deletion/addition? Somwthing like this:
	// https://stackoverflow.com/questions/63995289/how-to-remove-objects-from-golang-fyne-container
	l.Frame.RemoveAll()
	for i := 0; i < l.FrameSize; i++ {
		l.Frame.Add(l.List[pos+i].FrameColumn())
	}
	l.Frame.Refresh()

	l.FramePos = pos
}

// resize frame
func (l *PhotoList) resizeFrame(zoom int) {

	switch zoom {
	case RemoveColumn:
		if l.FrameSize-1 < MinFrameSize {
			return
		}
		l.List[l.FramePos+l.FrameSize-1].Img = nil
		l.FrameSize--
	case AddColumn:
		if l.FrameSize+1 > MaxFrameSize || l.FrameSize+1 > len(l.List) {
			return
		}
		i := l.FramePos + l.FrameSize
		if i == len(l.List) {
			l.FramePos--
			i = l.FramePos
		}
		l.List[i].Img = l.List[i].img(l.FrameSize)
		if l.List[i].Drop {
			l.List[i].Img.Translucency = 0.5
		}
		l.FrameSize++
	}
	//      0-1-2-3-4-5-6-7-8
	//          2-3-4			p=2, s=3
	// 		0-1-2				p=0, s=3
	// 					6-7-8	p=6, s=3

	// TODO: may be optimized when for scroll les than frame size by not all objects deletion/addition? Somwthing like this:
	// https://stackoverflow.com/questions/63995289/how-to-remove-objects-from-golang-fyne-container
	l.Frame.RemoveAll()
	for i := 0; i < l.FrameSize; i++ {
		l.Frame.Add(l.List[l.FramePos+i].FrameColumn())
	}
	l.Frame.Layout = layout.NewGridLayoutWithColumns(len(l.Frame.Objects))
	l.Frame.Refresh()
}

// fill frame Num photo images starting with Pos = 0.
func (l *PhotoList) initFrame() {
	l.FramePos = 0
	if l.FrameSize > len(l.List) {
		l.FrameSize = len(l.List)
	}
	for i := l.FramePos; i < l.FramePos+l.FrameSize && i < len(l.List); i++ {
		l.List[i].Img = l.List[i].img(l.FrameSize)
	}
}

// open photo folder dialog
func chooseFolder() {
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
		pl = newPhotoList(folder)
		MainLayout(pl)
	}, wMain)
	wd, _ := os.Getwd()
	savedLocation := fyne.CurrentApp().Preferences().StringWithFallback("folder", wd)
	locationUri, _ := storage.ListerForURI(storage.NewFileURI(savedLocation))
	fd.SetLocation(locationUri)
	fd.Resize(fyne.NewSize(672, 378))
	fd.Show()
}
