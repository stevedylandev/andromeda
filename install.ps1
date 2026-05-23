# Andromeda installer (Windows / PowerShell).
#
# Usage:
#   irm https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.ps1 | iex
#   # then:
#   Install-Andromeda sipp
#   Install-Andromeda feeds v0.4.0
#
# Or one-shot:
#   & ([scriptblock]::Create((irm https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.ps1))) sipp
#
# Env:
#   $env:INSTALL_DIR   target dir (default: $env:LOCALAPPDATA\andromeda)

param(
    [Parameter(Position = 0)][string]$App,
    [Parameter(Position = 1)][string]$Version
)

$ErrorActionPreference = "Stop"
$Repo = "stevedylandev/andromeda"

function Install-Andromeda {
    param(
        [Parameter(Mandatory = $true)][string]$App,
        [string]$Version
    )

    # Detect arch
    $archMap = @{
        "AMD64" = "x86_64"
        "ARM64" = "arm64"
    }
    $procArch = $env:PROCESSOR_ARCHITECTURE
    if (-not $archMap.ContainsKey($procArch)) {
        throw "Unsupported arch: $procArch"
    }
    $arch = $archMap[$procArch]
    if ($arch -eq "arm64") {
        throw "No windows/arm64 build published (goreleaser config ignores it)."
    }

    # Resolve version
    if (-not $Version) {
        Write-Host "Looking up latest $App release..."
        $releases = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases?per_page=50"
        $match = $releases | Where-Object { $_.tag_name -like "$App/v*" } | Select-Object -First 1
        if (-not $match) { throw "No releases found for $App" }
        $Version = ($match.tag_name -split "/", 2)[1]
    }

    $verNum  = $Version.TrimStart("v")
    $tag     = "$App/$Version"
    $archive = "${App}_${verNum}_windows_${arch}.zip"
    $url     = "https://github.com/$Repo/releases/download/$tag/$archive"

    $installDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "andromeda" }
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null

    $tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ([guid]::NewGuid()))
    try {
        Write-Host "Downloading $url"
        $zipPath = Join-Path $tmp $archive
        Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing
        Expand-Archive -Path $zipPath -DestinationPath $tmp -Force

        $exeName = "$App.exe"
        $src = Join-Path $tmp $exeName
        if (-not (Test-Path $src)) {
            throw "Binary $exeName not found in archive"
        }
        $dest = Join-Path $installDir $exeName
        Move-Item -Path $src -Destination $dest -Force

        Write-Host "Installed $App $Version to $dest"

        # PATH hint
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -notlike "*$installDir*") {
            Write-Host ""
            Write-Host "Add to PATH (run once):" -ForegroundColor Yellow
            Write-Host "  [Environment]::SetEnvironmentVariable('Path', `"`$env:Path;$installDir`", 'User')"
        }
    }
    finally {
        Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
    }
}

# If invoked with args (one-shot mode via scriptblock), run immediately.
if ($App) {
    Install-Andromeda -App $App -Version $Version
}
