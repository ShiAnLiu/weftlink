param(
    [Parameter(Mandatory = $true)][string]$PackageDir,
    [Parameter(Mandatory = $true)][string]$Version,
    [string]$OutFile = "",
    [string]$Publisher = "CN=Weftlink",
    [string]$CertificatePfx = "",
    [string]$CertificatePassword = "",
    [string]$MakeAppx = "",
    [string]$SignTool = ""
)

$ErrorActionPreference = "Stop"

function Get-FourPartVersion([string]$raw) {
    $v = $raw.Trim().TrimStart("v")
    $v = $v -replace '[^0-9.].*$', ''
    $parts = $v.Split(".")
    if ($parts.Count -eq 3) { $v = "$v.0" }
    $parts = $v.Split(".")
    if ($parts.Count -ne 4) { throw "MSIX version must have 4 numeric parts, got '$raw'" }
    foreach ($p in $parts) {
        if ($p -notmatch '^\d+$') { throw "MSIX version must be numeric, got '$raw'" }
    }
    return $v
}

function Find-SdkTool([string]$name) {
    $roots = @(
        (Join-Path ${env:ProgramFiles(x86)} "Windows Kits\10\bin"),
        (Join-Path $env:ProgramFiles "Windows Kits\10\bin")
    )
    $found = Get-ChildItem $roots -Recurse -Filter $name -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -match '\\x64\\' } |
        Sort-Object FullName -Descending |
        Select-Object -First 1
    if (-not $found) { throw "$name not found; install the Windows SDK" }
    return $found.FullName
}

function New-WeftlinkLogo([string]$path, [int]$size) {
    Add-Type -AssemblyName System.Drawing
    $bmp = New-Object System.Drawing.Bitmap($size, $size)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAlias
    $g.Clear([System.Drawing.Color]::FromArgb(79, 70, 229))
    $fontSize = [Math]::Max(10.0, $size * 0.46)
    $font = New-Object System.Drawing.Font("Segoe UI", $fontSize, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
    $brush = [System.Drawing.Brushes]::White
    $format = New-Object System.Drawing.StringFormat
    $format.Alignment = [System.Drawing.StringAlignment]::Center
    $format.LineAlignment = [System.Drawing.StringAlignment]::Center
    $g.DrawString("W", $font, $brush, (New-Object System.Drawing.RectangleF(0, 0, $size, $size)), $format)
    $g.Dispose()
    $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
    $bmp.Dispose()
}

$PackageDir = (Resolve-Path $PackageDir).Path
$msixVersion = Get-FourPartVersion $Version

$required = @("weftlink-gui.exe", "weftlink-tui.exe", "weftlink.exe", "weftlinkd.exe")
foreach ($name in $required) {
    if (-not (Test-Path (Join-Path $PackageDir $name))) {
        throw "missing $name in $PackageDir"
    }
}

$layout = Join-Path $PSScriptRoot "layout"
Remove-Item -Recurse -Force $layout -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Path $layout -Force | Out-Null
Copy-Item (Join-Path $PackageDir "*") $layout -Recurse -Force

$assets = Join-Path $layout "Assets"
New-Item -ItemType Directory -Path $assets -Force | Out-Null
New-WeftlinkLogo (Join-Path $assets "StoreLogo.png") 50
New-WeftlinkLogo (Join-Path $assets "Square44x44Logo.png") 44
New-WeftlinkLogo (Join-Path $assets "Square150x150Logo.png") 150

$escapedPublisher = [System.Security.SecurityElement]::Escape($Publisher)
$manifest = @"
<?xml version="1.0" encoding="utf-8"?>
<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
         xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
         xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities"
         IgnorableNamespaces="uap rescap">
  <Identity Name="Weftlink.Weftlink"
            Publisher="$escapedPublisher"
            Version="$msixVersion"
            ProcessorArchitecture="x64" />
  <Properties>
    <DisplayName>Weftlink</DisplayName>
    <PublisherDisplayName>Weftlink</PublisherDisplayName>
    <Logo>Assets\StoreLogo.png</Logo>
  </Properties>
  <Dependencies>
    <TargetDeviceFamily Name="Windows.Desktop" MinVersion="10.0.17763.0" MaxVersionTested="10.0.22621.0" />
  </Dependencies>
  <Resources>
    <Resource Language="en-us" />
  </Resources>
  <Applications>
    <Application Id="Weftlink" Executable="weftlink-gui.exe" EntryPoint="Windows.FullTrustApplication">
      <uap:VisualElements DisplayName="Weftlink"
                          Description="Weftlink desktop console"
                          BackgroundColor="transparent"
                          Square150x150Logo="Assets\Square150x150Logo.png"
                          Square44x44Logo="Assets\Square44x44Logo.png" />
    </Application>
  </Applications>
  <Capabilities>
    <rescap:Capability Name="runFullTrust" />
    <Capability Name="internetClient" />
    <Capability Name="privateNetworkClientServer" />
  </Capabilities>
</Package>
"@
[System.IO.File]::WriteAllText((Join-Path $layout "AppxManifest.xml"), $manifest, (New-Object System.Text.UTF8Encoding($false)))

if (-not $MakeAppx) { $MakeAppx = Find-SdkTool "makeappx.exe" }
if (-not $OutFile) { $OutFile = Join-Path $PSScriptRoot ("weftlink-{0}-x64.msix" -f $Version.TrimStart("v")) }
Remove-Item -Force $OutFile -ErrorAction SilentlyContinue

& $MakeAppx pack /d $layout /p $OutFile /o
if ($LASTEXITCODE -ne 0) { throw "makeappx failed with exit code $LASTEXITCODE" }

if ($CertificatePfx) {
    if (-not $SignTool) { $SignTool = Find-SdkTool "signtool.exe" }
    & $SignTool sign /fd SHA256 /f $CertificatePfx /p $CertificatePassword $OutFile
    if ($LASTEXITCODE -ne 0) { throw "signtool failed with exit code $LASTEXITCODE" }
}

Write-Output $OutFile
