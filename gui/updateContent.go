package gui

import (
	"runtime"
	"strconv"
	"time"
	"fmt"
	"strings"
	"slices"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/ShuvraneelMitra/hungry-daemons/managers"
)

const (
	uiFPS          = 30
	maxLogBytes    = 200_000
	maxMetricBytes = 100_000
)

func prettyByteSize(b uint64) string {
	bf := float64(b)
	for _, unit := range []string{"", "Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "Zi"} {
		if math.Abs(bf) < 1024.0 {
			return fmt.Sprintf("%3.1f %sB", bf, unit)
		}
		bf /= 1024.0
	}
	return fmt.Sprintf("%.1f YiB", bf)
}

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
			text := "Number of goroutines actively running = " + strconv.Itoa(numGoRoutines) + "\n"

			var memoryStats runtime.MemStats
			runtime.ReadMemStats(&memoryStats)

			text += "Bytes of allocated heap objects = " + prettyByteSize(memoryStats.HeapAlloc) + "\n"
			text += "Bytes of memory obtained from OS = " + prettyByteSize(memoryStats.Sys) + "\n"
			text += "Number of live heap objects = " + strconv.FormatUint(memoryStats.Mallocs - memoryStats.Frees, 10) + "\n"

			fyne.Do(func() {
				segment := layout.statusBar.body.Segments[0].(*widget.TextSegment)
				segment.Text = text
				layout.statusBar.body.Refresh()
			})
		}
	}()
}

func updateLogs(layout *guiLayout, msgChannel <-chan any) {
	go func() {
		ticker := time.NewTicker(time.Second / uiFPS)
		defer ticker.Stop()

		var buf strings.Builder
		logText := "Daemon logs...\n\n"

		for {
			select {
			case message, ok := <-msgChannel:
				if !ok {
					return
				}
				if s, ok := message.(string); ok {
					buf.WriteString(s)
					buf.WriteByte('\n')
				}

			case <-ticker.C:
				if buf.Len() == 0 {
					continue
				}

				chunk := buf.String()
				buf.Reset()

				fyne.Do(func() {
					logText += chunk
					if len(logText) > maxLogBytes {
						logText = logText[len(logText)-maxLogBytes:]
					}
					layout.logsView.SetText(logText)
					layout.logsView.CursorRow = len(layout.logsView.Text)
				})
			}
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
		ticker := time.NewTicker(time.Second / uiFPS)
		defer ticker.Stop()

		var latest []managers.LineageCount
		var rendered []managers.LineageCount
		var dirty bool

		sameLineageData := func(a, b []managers.LineageCount) bool {
			return slices.EqualFunc(a, b, func(x, y managers.LineageCount) bool {
				return x.Surname == y.Surname && x.Count == y.Count
			})
		}

		for {
			select {
			case data, ok := <-lineageChannel:
				if !ok {
					return
				}

				next, ok := data.([]managers.LineageCount)
				if !ok {
					continue
				}

				latest = slices.Clone(next)
				dirty = true

			case <-ticker.C:
				if !dirty {
					continue
				}

				if sameLineageData(latest, rendered) {
					dirty = false
					continue
				}

				snapshot := slices.Clone(latest)
				rendered = slices.Clone(latest)
				dirty = false

				fyne.Do(func() {
					layout.barPlot.SetData(snapshot)
				})
			}
		}
	}()
}
