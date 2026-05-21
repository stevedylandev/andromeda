module github.com/stevedylandev/andromeda/apps/feeds

go 1.25.0

require (
	github.com/mmcdole/gofeed v1.3.0
	github.com/stevedylandev/andromeda/pkg/auth v0.0.0
	github.com/stevedylandev/andromeda/pkg/config v0.0.0
	github.com/stevedylandev/andromeda/pkg/darkmatter v0.0.0
	github.com/stevedylandev/andromeda/pkg/sqlite v0.0.0
	github.com/stevedylandev/andromeda/pkg/web v0.0.0
	golang.org/x/crypto/x509roots/fallback v0.0.0-20260511143831-44decbfe70e2
	golang.org/x/net v0.41.0
)

replace (
	github.com/stevedylandev/andromeda/pkg/auth => ../../pkg/auth
	github.com/stevedylandev/andromeda/pkg/config => ../../pkg/config
	github.com/stevedylandev/andromeda/pkg/darkmatter => ../../pkg/darkmatter
	github.com/stevedylandev/andromeda/pkg/sqlite => ../../pkg/sqlite
	github.com/stevedylandev/andromeda/pkg/web => ../../pkg/web
)

require (
	github.com/PuerkitoBio/goquery v1.8.0 // indirect
	github.com/andybalholm/cascadia v1.3.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mmcdole/goxpp v1.1.1-0.20240225020742-a0c311522b23 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/exp v0.0.0-20250408133849-7e4ce0ab07d0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	modernc.org/libc v1.65.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.37.1 // indirect
)
