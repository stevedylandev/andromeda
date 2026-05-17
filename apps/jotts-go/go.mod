module github.com/stevedylandev/andromeda/apps/jotts-go

go 1.25.8

require (
	charm.land/bubbles/v2 v2.1.0
	charm.land/bubbletea/v2 v2.0.6
	charm.land/glamour/v2 v2.0.0
	charm.land/lipgloss/v2 v2.0.3
	github.com/BurntSushi/toml v1.6.0
	github.com/atotto/clipboard v0.1.4
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/stevedylandev/andromeda/crates-go/auth v0.0.0
	github.com/stevedylandev/andromeda/crates-go/config v0.0.0
	github.com/stevedylandev/andromeda/crates-go/darkmatter v0.0.0
	github.com/stevedylandev/andromeda/crates-go/sqlite v0.0.0
	github.com/stevedylandev/andromeda/crates-go/web v0.0.0
	github.com/yuin/goldmark v1.7.13
	golang.org/x/term v0.43.0
)

replace (
	github.com/stevedylandev/andromeda/crates-go/auth => ../../crates-go/auth
	github.com/stevedylandev/andromeda/crates-go/config => ../../crates-go/config
	github.com/stevedylandev/andromeda/crates-go/darkmatter => ../../crates-go/darkmatter
	github.com/stevedylandev/andromeda/crates-go/sqlite => ../../crates-go/sqlite
	github.com/stevedylandev/andromeda/crates-go/web => ../../crates-go/web
)

require (
	github.com/alecthomas/chroma/v2 v2.20.0 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260416155717-489999b90468 // indirect
	github.com/charmbracelet/x/ansi v0.11.7 // indirect
	github.com/charmbracelet/x/exp/slice v0.0.0-20250327172914-2fdc97757edf // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.23 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yuin/goldmark-emoji v1.0.6 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/exp v0.0.0-20250408133849-7e4ce0ab07d0 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	modernc.org/libc v1.65.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.37.1 // indirect
)
