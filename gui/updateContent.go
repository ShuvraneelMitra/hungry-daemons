package gui

import (
	"runtime"
	"strconv"
	"time"
	"fmt"

	"fyne.io/fyne/v2"
	"github.com/ShuvraneelMitra/hungry-daemons/managers"
)

func updateTime(layout *guiLayout) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			currentTime := time.Now().Format("2006-01-02 15:04:05")

			fyne.Do(func() {
				layout.header.right.Text = currentTime
				layout.header.right.Refresh()
			})
		}
	}()
}

func updateStatus(layout *guiLayout) {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			numGoRoutines := runtime.NumGoroutine()
			text := "Number of goroutines actively running = " + strconv.Itoa(numGoRoutines)

			fyne.Do(func() {
				layout.statusBar.body.Text = text
				layout.statusBar.body.Refresh()
			})
		}
	}()
}

func updateLogs(layout *guiLayout, msgChannel <-chan any) {
	go func() {
		for message := range msgChannel {
			msg := message

			fyne.Do(func() {
				if layout.logsView.Text == "Daemon logs...\n\n" {
					layout.logsView.SetText(msg.(string) + "\n")
				} else {
					layout.logsView.SetText(layout.logsView.Text + msg.(string) + "\n")
				}
				layout.logsView.CursorRow = len(layout.logsView.Text)
				layout.logsView.Refresh()
			})
		}
	}()
}

func updateGraph(layout *guiLayout, points <-chan any) {
	go func() {
		for p := range points {
			value := p

			fyne.Do(func() {
				layout.graph.AddPoint(value.(float64))
			})
		}
	}()
}

func updateMetrics(layout *guiLayout, metrics <-chan any){
	go func(){
		for metric := range metrics {
			fyne.Do(func(){
				layout.metricsView.SetText(layout.metricsView.Text + fmt.Sprint(metric))
			})
		}
	}()
}

func updateLineageGraph(layout *guiLayout, lineageChannel <-chan any) {
	go func() {
		for data := range lineageChannel {
			d := data.([]managers.LineageCount)

			fyne.Do(func() {
				layout.barPlot.SetData(d)
			})
		}
	}()
}
