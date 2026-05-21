module github.com/stevedylandev/andromeda/apps/og

go 1.24.4

require (
	github.com/stevedylandev/andromeda/crates-go/config v0.0.0
	github.com/stevedylandev/andromeda/crates-go/darkmatter v0.0.0
	github.com/stevedylandev/andromeda/crates-go/web v0.0.0
	golang.org/x/net v0.41.0
)

replace (
	github.com/stevedylandev/andromeda/crates-go/config => ../../crates-go/config
	github.com/stevedylandev/andromeda/crates-go/darkmatter => ../../crates-go/darkmatter
	github.com/stevedylandev/andromeda/crates-go/web => ../../crates-go/web
)
