module github.com/stevedylandev/andromeda/apps/og

go 1.24.4

require (
	github.com/stevedylandev/andromeda/pkg/config v0.0.0
	github.com/stevedylandev/andromeda/pkg/darkmatter v0.0.0
	github.com/stevedylandev/andromeda/pkg/web v0.0.0
	golang.org/x/net v0.41.0
)

replace (
	github.com/stevedylandev/andromeda/pkg/config => ../../pkg/config
	github.com/stevedylandev/andromeda/pkg/darkmatter => ../../pkg/darkmatter
	github.com/stevedylandev/andromeda/pkg/web => ../../pkg/web
)
