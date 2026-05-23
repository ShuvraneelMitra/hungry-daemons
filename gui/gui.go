package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

const (
	WINDOW_WIDTH = 1200
	WINDOW_HEIGHT = 615
)

func Run() {
	newApp := app.New()
	win := newApp.NewWindow("Hungry-Daemons")
	win.SetMaster()

	win.Resize(fyne.NewSize(WINDOW_WIDTH, WINDOW_HEIGHT))
	screen := newApp.Driver().AllWindows()[0]
	screen.CenterOnScreen()

	layout := getLayout()

	win.SetContent(layout.view)
	updateTime(layout)
	updateStatus(layout)

	win.ShowAndRun()
}
