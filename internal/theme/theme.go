package theme

import (
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type Palette struct {
	Bg        lipgloss.Color
	Panel     lipgloss.Color
	PanelAlt  lipgloss.Color
	Border    lipgloss.Color
	Text      lipgloss.Color
	Muted     lipgloss.Color
	Accent    lipgloss.Color
	AccentAlt lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Danger    lipgloss.Color
}

type Theme struct {
	Mode    string
	Profile termenv.Profile
	Palette Palette
	Styles  Styles
}

type Styles struct {
	StatusBar       lipgloss.Style
	StatusOK        lipgloss.Style
	StatusWarn      lipgloss.Style
	StatusDown      lipgloss.Style
	StatusBadgeOK   lipgloss.Style
	StatusBadgeWarn lipgloss.Style
	StatusBadgeDown lipgloss.Style
	Card            lipgloss.Style
	CardTitle       lipgloss.Style
	Muted           lipgloss.Style
	Accent          lipgloss.Style
	AccentAlt       lipgloss.Style
	Value           lipgloss.Style
	Border          lipgloss.Style
	KeyHint         lipgloss.Style
	PaneFocus       lipgloss.Style
	PaneHeader      lipgloss.Style
	Badge           lipgloss.Style
}

func Resolve(mode string) Theme {
	themeMode := strings.ToLower(strings.TrimSpace(mode))
	profile := termenv.EnvColorProfile()
	if themeMode != "dark" && themeMode != "light" {
		themeMode = detectBackground()
	}
	var palette Palette
	if themeMode == "dark" {
		palette = Palette{
			Bg:        pickColor(profile, "#050505", "232", "0"),
			Panel:     pickColor(profile, "#09090b", "233", "0"),
			PanelAlt:  pickColor(profile, "#18181b", "235", "0"),
			Border:    pickColor(profile, "#27272a", "237", "8"),
			Text:      pickColor(profile, "#fafafa", "255", "7"),
			Muted:     pickColor(profile, "#71717a", "243", "8"),
			Accent:    pickColor(profile, "#22d3ee", "45", "6"), // Cyan
			AccentAlt: pickColor(profile, "#10b981", "35", "2"), // Emerald
			Success:   pickColor(profile, "#4ade80", "120", "2"),
			Warning:   pickColor(profile, "#fbbf24", "214", "3"),
			Danger:    pickColor(profile, "#f87171", "203", "1"),
		}
	} else {
		palette = Palette{
			Bg:        pickColor(profile, "#f5f7fb", "255", "7"),
			Panel:     pickColor(profile, "#ffffff", "15", "7"),
			PanelAlt:  pickColor(profile, "#eef2f7", "254", "7"),
			Border:    pickColor(profile, "#d0d7e2", "250", "8"),
			Text:      pickColor(profile, "#1a2233", "235", "0"),
			Muted:     pickColor(profile, "#6b7280", "243", "8"),
			Accent:    pickColor(profile, "#0f766e", "30", "6"),
			AccentAlt: pickColor(profile, "#2563eb", "27", "4"),
			Success:   pickColor(profile, "#059669", "35", "2"),
			Warning:   pickColor(profile, "#d97706", "172", "3"),
			Danger:    pickColor(profile, "#dc2626", "160", "1"),
		}
	}
	return Theme{
		Mode:    themeMode,
		Profile: profile,
		Palette: palette,
		Styles:  buildStyles(palette),
	}
}

func buildStyles(p Palette) Styles {
	badgeBase := lipgloss.NewStyle().
		Foreground(p.Panel).
		Bold(true).
		Padding(0, 1)

	return Styles{
		StatusBar: lipgloss.NewStyle().
			Background(p.Bg).
			Foreground(p.Muted).
			Padding(0, 1),
		StatusOK:        lipgloss.NewStyle().Foreground(p.Success).Bold(true),
		StatusWarn:      lipgloss.NewStyle().Foreground(p.Warning).Bold(true),
		StatusDown:      lipgloss.NewStyle().Foreground(p.Danger).Bold(true),
		StatusBadgeOK:   badgeBase.Background(p.Success),
		StatusBadgeWarn: badgeBase.Background(p.Warning),
		StatusBadgeDown: badgeBase.Background(p.Danger),
		Card: lipgloss.NewStyle().
			Background(p.Panel).
			Foreground(p.Text).
			Padding(1, 2).
			Border(lipgloss.NormalBorder(), false, false, false, true). // Left border only for cleaner look
			BorderForeground(p.Border),
		CardTitle: lipgloss.NewStyle().Foreground(p.Accent).Bold(true),
		Muted:     lipgloss.NewStyle().Foreground(p.Muted),
		Accent:    lipgloss.NewStyle().Foreground(p.Accent).Bold(true),
		AccentAlt: lipgloss.NewStyle().Foreground(p.AccentAlt).Bold(true),
		Value:     lipgloss.NewStyle().Foreground(p.Text).Bold(true),
		Border:    lipgloss.NewStyle().Foreground(p.Border),
		KeyHint: lipgloss.NewStyle().
			Foreground(p.Muted).
			Padding(0, 1),
		PaneFocus: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(p.Accent),
		PaneHeader: lipgloss.NewStyle().
			Foreground(p.Accent).
			Bold(true).
			Padding(0, 1),
		Badge: lipgloss.NewStyle().
			Foreground(p.Panel).
			Background(p.Accent).
			Padding(0, 1),
	}
}

func detectBackground() string {
	if val := os.Getenv("COLORFGBG"); val != "" {
		parts := strings.Split(val, ";")
		bg := parts[len(parts)-1]
		if num, err := strconv.Atoi(bg); err == nil {
			if num <= 6 || num == 8 {
				return "dark"
			}
			return "light"
		}
	}
	return "dark"
}

func pickColor(profile termenv.Profile, hex, ansi256, ansi string) lipgloss.Color {
	switch profile {
	case termenv.TrueColor:
		return lipgloss.Color(hex)
	case termenv.ANSI256:
		return lipgloss.Color(ansi256)
	default:
		return lipgloss.Color(ansi)
	}
}
