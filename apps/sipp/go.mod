module github.com/stevedylandev/andromeda/apps/sipp

go 1.25.0

require (
	charm.land/bubbles/v2 v2.1.0
	charm.land/bubbletea/v2 v2.0.6
	charm.land/lipgloss/v2 v2.0.3
	github.com/alecthomas/chroma/v2 v2.24.1
	github.com/atotto/clipboard v0.1.4
	github.com/stevedylandev/andromeda/pkg/auth v0.0.0
	github.com/stevedylandev/andromeda/pkg/config v0.0.0
	github.com/stevedylandev/andromeda/pkg/darkmatter v0.0.0
	github.com/stevedylandev/andromeda/pkg/sqlite v0.0.0
	github.com/stevedylandev/andromeda/pkg/tui v0.0.0
	github.com/stevedylandev/andromeda/pkg/web v0.0.0
	golang.org/x/term v0.43.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260416155717-489999b90468 // indirect
	github.com/charmbracelet/x/ansi v0.11.7 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/dlclark/regexp2 v1.12.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.23 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sahilm/fuzzy v0.1.1 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/exp v0.0.0-20250408133849-7e4ce0ab07d0 // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	modernc.org/libc v1.65.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.37.1 // indirect
)

replace (
	github.com/stevedylandev/andromeda/pkg/auth => ../../pkg/auth
	github.com/stevedylandev/andromeda/pkg/config => ../../pkg/config
	github.com/stevedylandev/andromeda/pkg/darkmatter => ../../pkg/darkmatter
	github.com/stevedylandev/andromeda/pkg/sqlite => ../../pkg/sqlite
	github.com/stevedylandev/andromeda/pkg/tui => ../../pkg/tui
	github.com/stevedylandev/andromeda/pkg/web => ../../pkg/web
)
