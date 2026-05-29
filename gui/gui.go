package gui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

const (
	WINDOW_WIDTH = 1200
	WINDOW_HEIGHT = 615
)

func updateContent(channels map[string]chan any, layout *guiLayout) {
	updateTime(layout)
	updateStatus(layout)
	updateLogs(layout, channels["logs"])
	updateGraph(layout, channels["populationData"])
	updateMetrics(layout, channels["metrics"])
	updateLineageGraph(layout, channels["topK"])
}

func Run(channels map[string]chan any, cancel context.CancelFunc) {
	newApp := app.New()
	win := newApp.NewWindow("Hungry-Daemons")
	win.SetMaster()
	win.SetIcon(resourceIcon)

	win.Resize(fyne.NewSize(WINDOW_WIDTH, WINDOW_HEIGHT))
	screen := newApp.Driver().AllWindows()[0]
	screen.CenterOnScreen()

	layout := getLayout()

	win.SetContent(layout.view)
	updateContent(channels, layout)

	win.SetOnClosed(func() {
		cancel()
	})

	win.ShowAndRun()
}
