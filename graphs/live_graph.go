package graphs

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

var LAVENDER = color.NRGBA{R: 167, G: 143, B: 255, A: 255}

type LiveGraph struct {
	widget.BaseWidget

	values    []float64
	maxPoints int
}

func NewLiveGraph(maxPoints int) *LiveGraph {
	g := &LiveGraph{
		values:    make([]float64, 0, maxPoints),
		maxPoints: maxPoints,
	}

	g.ExtendBaseWidget(g)
	return g
}

func (g *LiveGraph) AddPoint(y float64) {
	g.values = append(g.values, y)
	g.Refresh()
}

func (g *LiveGraph) CreateRenderer() fyne.WidgetRenderer {
	r := &liveGraphRenderer{
		graph: g,
		bg:    canvas.NewRectangle(color.NRGBA{R: 10, G: 10, B: 10, A: 255}),
	}

	r.Refresh()
	return r
}

type liveGraphRenderer struct {
	graph   *LiveGraph
	bg      *canvas.Rectangle
	objects []fyne.CanvasObject
}

func (r *liveGraphRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.Refresh()
}

func (r *liveGraphRenderer) MinSize() fyne.Size {
	return fyne.NewSize(400, 250)
}

func (r *liveGraphRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *liveGraphRenderer) Destroy() {}

func (r *liveGraphRenderer) Refresh() {
	values := r.graph.values
	size := r.graph.Size()

	r.bg.Resize(size)
	r.objects = []fyne.CanvasObject{r.bg}

	if len(values) < 2 {
		return
	}

	width := size.Width
	height := size.Height

	leftPad := float32(55)
	rightPad := float32(20)
	topPad := float32(20)
	bottomPad := float32(35)

	plotWidth := width - leftPad - rightPad
	plotHeight := height - topPad - bottomPad

	if plotWidth <= 0 || plotHeight <= 0 {
		return
	}

	minY := values[0]
	maxY := values[0]

	for _, v := range values {
		if v < minY {
			minY = v
		}
		if v > maxY {
			maxY = v
		}
	}

	if maxY == minY {
		maxY = minY + 1
	}

	x0 := leftPad
	y0 := topPad + plotHeight
	x1 := leftPad + plotWidth

	xAxis := canvas.NewLine(color.White)
	xAxis.StrokeWidth = 1
	xAxis.Position1 = fyne.NewPos(x0, y0)
	xAxis.Position2 = fyne.NewPos(x1, y0)

	yAxis := canvas.NewLine(color.White)
	yAxis.StrokeWidth = 1
	yAxis.Position1 = fyne.NewPos(x0, topPad)
	yAxis.Position2 = fyne.NewPos(x0, y0)

	r.objects = append(r.objects, xAxis, yAxis)

	yTickCount := 5
	for i := 0; i <= yTickCount; i++ {
		t := float64(i) / float64(yTickCount)
		value := minY + t*(maxY-minY)

		y := y0 - float32(t)*plotHeight

		tick := canvas.NewLine(color.White)
		tick.StrokeWidth = 1
		tick.Position1 = fyne.NewPos(x0-5, y)
		tick.Position2 = fyne.NewPos(x0, y)

		label := canvas.NewText(fmt.Sprintf("%.1f", value), color.White)
		label.TextSize = 10
		label.TextStyle = fyne.TextStyle{Monospace: true}
		label.Move(fyne.NewPos(5, y-7))

		r.objects = append(r.objects, tick, label)
	}

	xTickCount := 5
	for i := 0; i <= xTickCount; i++ {
		t := float32(i) / float32(xTickCount)
		index := int(t * float32(len(values)-1))

		x := x0 + t*plotWidth

		tick := canvas.NewLine(color.White)
		tick.StrokeWidth = 1
		tick.Position1 = fyne.NewPos(x, y0)
		tick.Position2 = fyne.NewPos(x, y0+5)

		label := canvas.NewText(fmt.Sprintf("%d", index), color.White)
		label.TextSize = 10
		label.TextStyle = fyne.TextStyle{Monospace: true}
		label.Move(fyne.NewPos(x-8, y0+8))

		r.objects = append(r.objects, tick, label)
	}

	stepX := plotWidth / float32(len(values)-1)

	for i := 0; i < len(values)-1; i++ {
		xA := x0 + float32(i)*stepX
		xB := x0 + float32(i+1)*stepX

		yA := y0 - float32((values[i]-minY)/(maxY-minY))*plotHeight
		yB := y0 - float32((values[i+1]-minY)/(maxY-minY))*plotHeight

		line := canvas.NewLine(LAVENDER)
		line.StrokeWidth = 2
		line.Position1 = fyne.NewPos(xA, yA)
		line.Position2 = fyne.NewPos(xB, yB)

		r.objects = append(r.objects, line)
	}
}
