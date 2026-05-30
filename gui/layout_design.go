package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ShuvraneelMitra/hungry-daemons/graphs"
)

type Header struct {
	left  *canvas.Text
	right *canvas.Text

	view  *container.ThemeOverride
}

type StatusBar struct {
	title *canvas.Text
	body *widget.RichText

	view *fyne.Container
}

type guiLayout struct {
	sidebar fyne.CanvasObject
	chartArea *widget.Label
	statusBar *StatusBar
	header *Header
	footer fyne.CanvasObject

	graph *graphs.LiveGraph
	barPlot *graphs.LiveBarGraph

	tabs *container.AppTabs
	logsView  *widget.Entry
	metricsView *widget.Entry

	view *fyne.Container
}

func getHeader(staticText string) *Header {
	left := canvas.NewText(
		staticText,
		theme.ForegroundColor(),
	)
	left.TextStyle = fyne.TextStyle{Monospace: true}
	left.TextSize = theme.TextSize() * 0.75

	right := canvas.NewText(
		"",
		theme.ForegroundColor(),
	)
	right.TextStyle = fyne.TextStyle{Monospace: true}
	right.TextSize = theme.TextSize() * 0.75

	content := container.NewBorder(
		nil,  
		nil,  
		left, 
		right,
		nil,
	)

	return &Header{
		left:  left,
		right: right,
		view:  container.NewThemeOverride(content, NewBorderTheme()),
	}
}

func getStatusBar() *StatusBar {
	title := canvas.NewText("Statistics", LAVENDER)
	title.TextStyle = fyne.TextStyle{
		Bold:      true,
		Underline: true,
	}
	title.TextSize = theme.TextSize() * HEADING_TO_BODY

	bodyText := widget.NewRichText(&widget.TextSegment{
		Style: widget.RichTextStyle{
			SizeName: theme.SizeNameCaptionText, 
		},
		Text: "Default Body",
	})

	content := container.NewBorder(
		title, 
		nil,   
		nil,   
		nil,   
		bodyText, 
	)

	return &StatusBar{
		title: title,
		body:  bodyText, 
		view:  content,
	}
}

func getLayout() *guiLayout {
	header := getHeader("hungry-daemons: an exploration of goroutines by insipidintegrator")

	footer := func() fyne.CanvasObject {
		t := canvas.NewText(
			"\u00A9 ShuvraneelMitra",
			theme.Color(theme.ColorNameForeground),
		)
		t.TextStyle = fyne.TextStyle{Monospace: true}
		t.TextSize = theme.TextSize() * 0.75
		return t
	}()

	themedFooter := getThemedHeaderandFooter(footer)

    lineageGraph := graphs.NewLiveBarGraph(5)

	sidebarTitle := canvas.NewText("Top Lineages", GOPHER)
	sidebarTitle.TextStyle = fyne.TextStyle{
		Bold:      true,
		Monospace: true,
	}
	sidebarTitle.TextSize = theme.TextSize() * 0.9
	leftPad := canvas.NewRectangle(color.Transparent)
	leftPad.SetMinSize(fyne.NewSize(6, 0))
	topPad := canvas.NewRectangle(color.Transparent)
	topPad.SetMinSize(fyne.NewSize(0, 10))

	paddedTitle := container.NewBorder(
		topPad,
		nil,
		leftPad,
		nil,
		sidebarTitle,
	)

	sidebarContent := container.NewBorder(
		paddedTitle,
		nil,
		nil,
		nil,
		container.NewMax(lineageGraph),
	)

	themedSidebar := getThemedSidebar(sidebarContent)

    top := widget.NewLabel("MainWindow")
	bottom := getStatusBar()

	graph := graphs.NewLiveGraph(200)
	dashboardView := container.NewMax(graph)

	logsView := widget.NewMultiLineEntry()
	logsView.SetText("Daemon logs...\n\n")
	logsView.TextStyle = fyne.TextStyle{
		Monospace: true,
	}

	metricsView := widget.NewMultiLineEntry()
	metricsView.TextStyle = fyne.TextStyle{
		Monospace: true,
	}

	tabs := container.NewAppTabs(
		container.NewTabItem("Dashboard", dashboardView),
		container.NewTabItem("Logs", logsView),
		container.NewTabItem("Metrics", metricsView),
	)

	tabs.SetTabLocation(container.TabLocationTop)
	themedTabs := getThemedTabs(tabs)

	mainWin := container.NewVSplit(
		themedTabs,
		container.NewPadded(bottom.view),
	)
	mainWin.Offset = 0.85

	split := container.NewHSplit(
		themedSidebar,
		mainWin,
	)
	split.Offset = 0.15

    compiledContainer := container.NewBorder(
        header.view.Content, 
        themedFooter, 
        nil,   
        nil,    
        split, 
    )

	return &guiLayout {
		sidebar: themedSidebar,
		chartArea: top,
		statusBar: bottom,
		view: compiledContainer,
		header: header,
		footer: themedFooter,
		graph: graph,
		barPlot: lineageGraph,
		tabs: tabs,
		logsView: logsView,
		metricsView: metricsView,
	}
}
