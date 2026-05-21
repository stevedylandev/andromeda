module github.com/stevedylandev/andromeda/apps/posts

go 1.24.4

require (
	github.com/stevedylandev/andromeda/pkg/auth v0.0.0
	github.com/stevedylandev/andromeda/pkg/config v0.0.0
	github.com/stevedylandev/andromeda/pkg/darkmatter v0.0.0
	github.com/stevedylandev/andromeda/pkg/sqlite v0.0.0
	github.com/stevedylandev/andromeda/pkg/web v0.0.0
	github.com/yuin/goldmark v1.7.8
)

require (
	github.com/aws/aws-sdk-go-v2 v1.41.7 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.10 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.101.0 // indirect
	github.com/aws/smithy-go v1.25.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/exp v0.0.0-20250408133849-7e4ce0ab07d0 // indirect
	golang.org/x/sys v0.33.0 // indirect
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
	github.com/stevedylandev/andromeda/pkg/web => ../../pkg/web
)
