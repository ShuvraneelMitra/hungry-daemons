package gui

import (
	"runtime"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
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

func updateLogs(layout *guiLayout, msgChannel <-chan string) {
	go func() {
		for message := range msgChannel {
			msg := message

			fyne.Do(func() {
				if layout.logsView.Text == "Daemon logs...\n\n" {
					layout.logsView.SetText(msg + "\n")
				} else {
					layout.logsView.SetText(layout.logsView.Text + msg + "\n")
				}
				layout.logsView.CursorRow = len(layout.logsView.Text)
				layout.logsView.Refresh()
			})
		}
	}()
}
