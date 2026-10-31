param(
    [Parameter(Mandatory = $true)][string]$QtDir,
    [string]$OpenSslDir = "",
    [string]$MinGwBin = "",
    [string]$BuildDir = "",
    [string]$OutDir = ""
)

$ErrorActionPreference = "Stop"

if (-not $BuildDir) {
    $BuildDir = Join-Path $PSScriptRoot "..\..\..\build\cpp"
}
if (-not $OutDir) {
    $OutDir = Join-Path $BuildDir "dist"
}
if (-not $MinGwBin) {
    $gxx = Get-Command "g++.exe" -ErrorAction SilentlyContinue
    if ($gxx) {
        $MinGwBin = Split-Path $gxx.Source -Parent
    }
}

$BuildDir = (Resolve-Path $BuildDir).Path
$windeployqt = Join-Path $QtDir "bin\windeployqt.exe"
if (-not (Test-Path $windeployqt)) {
    throw "windeployqt not found: $windeployqt"
}

$env:PATH = (Join-Path $QtDir "bin") + ";" + $env:PATH
if ($MinGwBin) {
    $env:PATH = $MinGwBin + ";" + $env:PATH
}
if ($OpenSslDir) {
    $env:PATH = $OpenSslDir + ";" + $env:PATH
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$gui = Join-Path $BuildDir "weftlink-gui.exe"
$tui = Join-Path $BuildDir "weftlink-tui.exe"
if (-not (Test-Path $gui) -or -not (Test-Path $tui)) {
    throw "build first: cmake --build $BuildDir"
}

Copy-Item $gui $OutDir -Force
Copy-Item $tui $OutDir -Force

$deployArgs = @("--release", "--no-translations", "--compiler-runtime", "--dir", $OutDir,
                (Join-Path $OutDir "weftlink-gui.exe"),
                (Join-Path $OutDir "weftlink-tui.exe"))
$previous = $ErrorActionPreference
$ErrorActionPreference = "Continue"
& $windeployqt @deployArgs
$deployExit = $LASTEXITCODE
$ErrorActionPreference = $previous
if ($deployExit -ne 0) {
    throw "windeployqt failed with exit code $deployExit"
}

if ($OpenSslDir) {
    foreach ($dll in @("libssl-3-x64.dll", "libcrypto-3-x64.dll")) {
        $source = Join-Path $OpenSslDir $dll
        if (Test-Path $source) {
            Copy-Item $source $OutDir -Force
        }
    }
}

if ($MinGwBin) {
    foreach ($dll in @("libstdc++-6.dll", "libgcc_s_seh-1.dll", "libwinpthread-1.dll")) {
        $source = Join-Path $MinGwBin $dll
        if (Test-Path $source) {
            Copy-Item $source $OutDir -Force
        }
    }
}

Write-Host "Deployed weftlink-gui.exe and weftlink-tui.exe to $OutDir"
