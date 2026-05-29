package graphs

import (
	"fmt"
	"image/color"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/ShuvraneelMitra/hungry-daemons/managers"
)

type LiveBarGraph struct {
	widget.BaseWidget

	items []managers.LineageCount
	limit int
}

var BarColors = []color.NRGBA{
	{R: 167, G: 143, B: 255, A: 255}, // Lavender
	{R: 0, G: 255, B: 255, A: 255},   // Cyan
	{R: 245, G: 193, B: 108, A: 255}, // Amber
	{R: 255, G: 105, B: 180, A: 255}, // Pink
	{R: 50, G: 205, B: 50, A: 255},   // Lime
}

func NewLiveBarGraph(limit int) *LiveBarGraph {
	g := &LiveBarGraph{
		items: make([]managers.LineageCount, 0, limit),
		limit: limit,
	}
	g.ExtendBaseWidget(g)
	return g
}

func (g *LiveBarGraph) SetData(items []managers.LineageCount) {
	copied := append([]managers.LineageCount(nil), items...)

	sort.Slice(copied, func(i, j int) bool {
		return copied[i].Count > copied[j].Count
	})

	if len(copied) > g.limit {
		copied = copied[:g.limit]
	}

	g.items = copied
	g.Refresh()
}

func (g *LiveBarGraph) CreateRenderer() fyne.WidgetRenderer {
	r := &liveBarGraphRenderer{
		graph: g,
		bg:    canvas.NewRectangle(color.NRGBA{R: 10, G: 10, B: 10, A: 255}),
	}
	r.Refresh()
	return r
}

type liveBarGraphRenderer struct {
	graph   *LiveBarGraph
	bg      *canvas.Rectangle
	objects []fyne.CanvasObject
}

func (r *liveBarGraphRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.Refresh()
}

func (r *liveBarGraphRenderer) MinSize() fyne.Size {
	return fyne.NewSize(400, 250)
}

func (r *liveBarGraphRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *liveBarGraphRenderer) Destroy() {}

func (r *liveBarGraphRenderer) Refresh() {
	size := r.graph.Size()

	r.bg.Resize(size)
	r.objects = []fyne.CanvasObject{r.bg}

	items := r.graph.items
	if len(items) == 0 {
		return
	}

	width := size.Width
	height := size.Height

	leftPad := float32(12)
	rightPad := float32(35)
	topPad := float32(10)
	bottomPad := float32(20)

	plotWidth := width - leftPad - rightPad
	plotHeight := height - topPad - bottomPad

	if plotWidth <= 0 || plotHeight <= 0 {
		return
	}

	maxCount := items[0].Count
	if maxCount <= 0 {
		maxCount = 1
	}

	rowGap := float32(10)
	barHeight := float32(13)

	for i, item := range items {
		y := topPad + float32(i)*(barHeight+rowGap)

		label := canvas.NewText(item.Surname, color.White)
		label.TextStyle = fyne.TextStyle{Monospace: true, Bold: true,}
		label.Move(fyne.NewPos(leftPad,	y))
		label.TextSize = 11

		barWidth := float32(item.Count) / float32(maxCount) * plotWidth * 0.75

		bar := canvas.NewRectangle(
			BarColors[i%len(BarColors)],
		)
		bar.Resize(fyne.NewSize(barWidth, barHeight))
		bar.Move(fyne.NewPos(leftPad + 70, y))

		value := canvas.NewText(fmt.Sprintf("%d", item.Count), color.White)
		value.TextSize = 11
		value.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
		value.Move(fyne.NewPos(leftPad+barWidth+80, y+(barHeight/2)-6))

		r.objects = append(r.objects, label, bar, value)
	}
}
