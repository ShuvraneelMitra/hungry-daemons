package gui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
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

func Run(from map[string]chan any, to map[string]chan any, cancel context.CancelFunc) {
	newApp := app.New()
	win := newApp.NewWindow("Hungry-Daemons")
	win.SetMaster()
	win.SetIcon(resourceIcon)

	win.Resize(fyne.NewSize(WINDOW_WIDTH, WINDOW_HEIGHT))
	screen := newApp.Driver().AllWindows()[0]
	screen.CenterOnScreen()

	layout := getLayout()
	layout.statusBar.SetShutDownButtonFunc(func() {
		to["shutdown"]<-struct{}{}

		for _, channel := range to {
			close(channel)
		}
	})

	layout.statusBar.SetPauseButtonFunc(func() {
		paused = !paused

		if paused {
			to["pause"]<-struct{}{}
			controlButton.SetText("Resume")
			controlButton.SetIcon(theme.MediaPlayIcon())
		} else {
			to["pause"]<-struct{}{}
			controlButton.SetText("Pause")
			controlButton.SetIcon(theme.MediaPauseIcon())
		}
	})

	win.SetContent(layout.view)
	updateContent(from, layout)

	win.SetOnClosed(func() {
		cancel()
	})

	win.ShowAndRun()
}
