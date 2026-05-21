module github.com/stevedylandev/andromeda/apps/shrink

go 1.24.4

require (
	github.com/stevedylandev/andromeda/crates-go/config v0.0.0
	github.com/stevedylandev/andromeda/crates-go/darkmatter v0.0.0
	github.com/stevedylandev/andromeda/crates-go/web v0.0.0
	golang.org/x/image v0.27.0
)

replace (
	github.com/stevedylandev/andromeda/crates-go/config => ../../crates-go/config
	github.com/stevedylandev/andromeda/crates-go/darkmatter => ../../crates-go/darkmatter
	github.com/stevedylandev/andromeda/crates-go/web => ../../crates-go/web
)
