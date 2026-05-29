package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

const (
	HEADING_TO_BODY = 1.2
)
var (
	LAVENDER = color.NRGBA{R: 167, G: 143, B: 255, A: 255}
	GOPHER = color.NRGBA{R : 37, G : 245, B : 252, A : 255}
)

//------------------------------------------ SIDEBAR THEME -------------------------------------------------
type SidebarTheme struct {
}

func (t *SidebarTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		if variant == theme.VariantLight {
			return color.RGBA{R: 255, G: 255, B: 255, A: 255} 
		}
		return color.RGBA{R: 30, G: 30, B: 30, A: 255}
	
	case theme.ColorNameForeground:
		if variant == theme.VariantLight {
			return color.RGBA{R: 0, G: 0, B: 0, A: 255}
		}
		return color.RGBA{R: 0, G: 255, B: 255, A: 255}

	case theme.ColorNameButton:
		return color.RGBA{R: 0, G: 122, B: 255, A: 255} 
	
	case theme.ColorNameDisabled:
		return color.RGBA{R: 128, G: 128, B: 128, A: 255}
	
	case theme.ColorNamePrimary:
		return color.RGBA{R: 0, G: 122, B: 255, A: 255} 
	
	case theme.ColorNameHover:
		return color.RGBA{R: 200, G: 200, B: 200, A: 30}
	
	case theme.ColorNameFocus:
		return color.RGBA{R: 0, G: 122, B: 255, A: 128} 
	
	case theme.ColorNameShadow:
		return color.RGBA{R: 0, G: 0, B: 0, A: 64}
	
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *SidebarTheme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Bold {
		if style.Italic {
			return theme.DefaultTheme().Font(style) 
		}
		return theme.DefaultTheme().Font(style) 
	}
	if style.Italic {
		return theme.DefaultTheme().Font(style) 
	}
	if style.Monospace {
		return theme.DefaultTheme().Font(style)
	}
	return theme.DefaultTheme().Font(style)
}

func (t *SidebarTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *SidebarTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInlineIcon:
		return 20
	case theme.SizeNameScrollBar:
		return 16
	case theme.SizeNameScrollBarSmall:
		return 3
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 24
	case theme.SizeNameSubHeadingText:
		return 18
	case theme.SizeNameCaptionText:
		return 11
	case theme.SizeNameInputBorder:
		return 2
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func NewSidebarTheme() fyne.Theme {
	return &SidebarTheme{}
}

func getThemedSidebar(obj fyne.CanvasObject) *container.ThemeOverride {
	theme := NewSidebarTheme()
	return container.NewThemeOverride(obj, theme)
}

//------------------------------------------ HEADER/FOOTER THEME -------------------------------------------------
type BorderTheme struct {
}

func (t *BorderTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		if variant == theme.VariantLight {
			return color.RGBA{R: 255, G: 255, B: 255, A: 255} 
		}
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}
	
	case theme.ColorNameForeground:
		if variant == theme.VariantLight {
			return color.RGBA{R: 0, G: 0, B: 0, A: 255}
		}
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *BorderTheme) Font(style fyne.TextStyle) fyne.Resource {
	if style.Bold {
		if style.Italic {
			return theme.DefaultTheme().Font(style) 
		}
		return theme.DefaultTheme().Font(style) 
	}
	if style.Italic {
		return theme.DefaultTheme().Font(style) 
	}
	if style.Monospace {
		return theme.DefaultTheme().Font(style)
	}
	return theme.DefaultTheme().Font(style)
}

func (t *BorderTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *BorderTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name) * 0.75
}

func NewBorderTheme() fyne.Theme {
	return &BorderTheme{}
}

func getThemedHeaderandFooter(obj fyne.CanvasObject) *container.ThemeOverride {
	theme := NewBorderTheme()
	return container.NewThemeOverride(obj, theme)
}

// --------------------------------- TABS THEME --------------------------------

type TabsTheme struct{}

func (t *TabsTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.RGBA{R: 0, G: 0, B: 0, A: 255}

	case theme.ColorNameForeground:
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}

	case theme.ColorNamePrimary:
		return LAVENDER

	case theme.ColorNameButton:
		return color.RGBA{R: 30, G: 30, B: 30, A: 255}

	case theme.ColorNameHover:
		return color.RGBA{R: 167, G: 143, B: 255, A: 40}

	case theme.ColorNameFocus:
		return color.RGBA{R: 167, G: 143, B: 255, A: 120}

	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t *TabsTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *TabsTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *TabsTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 2
	case theme.SizeNameText:
		return 12
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func NewTabsTheme() fyne.Theme {
	return &TabsTheme{}
}

func getThemedTabs(obj fyne.CanvasObject) *container.ThemeOverride {
	return container.NewThemeOverride(obj, NewTabsTheme())
}
