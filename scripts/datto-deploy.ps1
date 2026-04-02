<#
.SYNOPSIS
    Datto RMM Component: Deploy LGP MCP Servers
.DESCRIPTION
    Downloads and installs LGP MCP server binaries from GitHub Releases.
    Detects OS/Arch automatically, downloads the correct binary, and installs
    to the specified path. Designed to run as a Datto RMM Component.
.PARAMETER Version
    Release version to install (e.g., "v1.0.0"). Defaults to "latest".
.PARAMETER Servers
    Comma-separated list of servers to install. Defaults to all.
.PARAMETER InstallPath
    Installation directory. Defaults to "C:\Program Files\LGP-MCP" on Windows,
    "/usr/local/bin" on macOS/Linux.
#>

param(
    [string]$Version = "latest",
    [string]$Servers = "autotask-mcp,itglue-mcp,datto-rmm-mcp,rocketcyber-mcp",
    [string]$InstallPath = ""
)

$ErrorActionPreference = "Stop"
$repo = "Logiphys/lgp-mcp-servers"

# Detect platform
if ($IsWindows -or $env:OS -eq "Windows_NT") {
    $os = "windows"
    $ext = ".exe"
    if (-not $InstallPath) { $InstallPath = "C:\Program Files\LGP-MCP" }
} elseif ($IsMacOS) {
    $os = "darwin"
    $ext = ""
    if (-not $InstallPath) { $InstallPath = "/usr/local/bin" }
} else {
    $os = "linux"
    $ext = ""
    if (-not $InstallPath) { $InstallPath = "/usr/local/bin" }
}

# Detect architecture
$arch = if ([Environment]::Is64BitOperatingSystem) {
    if ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq [System.Runtime.InteropServices.Architecture]::Arm64) {
        "arm64"
    } else {
        "amd64"
    }
} else {
    "amd64"
}

Write-Host "Platform: $os/$arch"
Write-Host "Install path: $InstallPath"

# Resolve version
if ($Version -eq "latest") {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
    $Version = $release.tag_name
}
Write-Host "Version: $Version"

# Ensure install directory exists
if (-not (Test-Path $InstallPath)) {
    New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
}

# Download and install each server
$serverList = $Servers -split ","
foreach ($server in $serverList) {
    $server = $server.Trim()
    $binaryName = "${server}-${os}-${arch}${ext}"
    $downloadUrl = "https://github.com/$repo/releases/download/$Version/$binaryName"
    $destPath = Join-Path $InstallPath "${server}${ext}"

    Write-Host "Downloading $binaryName..."
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $destPath -UseBasicParsing
        if ($os -ne "windows") {
            chmod +x $destPath
        }
        Write-Host "  Installed: $destPath"
    } catch {
        Write-Warning "  Failed to download $binaryName : $_"
    }
}

Write-Host "`nInstallation complete. Configure MCP clients with the binary paths above."
