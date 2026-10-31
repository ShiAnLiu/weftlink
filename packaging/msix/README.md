# MSIX packaging

CI (`qt-windows`) and CD (`msix` job in `release.yml`) build a signed
`weftlink-<version>-x64.msix` next to the normal zip/tar assets.

## What goes in the package

- `weftlink-gui.exe`, `weftlink-tui.exe` (Qt/OpenSSL/MSVC runtime deployed by
  `windeployqt`)
- `weftlink.exe`, `weftlinkd.exe`
- `AppxManifest.xml` + generated Assets (`StoreLogo`, 44x44, 150x150)
- Full-trust desktop app with `internetClient` and
  `privateNetworkClientServer` capabilities; the GUI is the Start-menu entry.

## Local build

Requires the Windows SDK (`makeappx.exe`, `signtool.exe`) and a directory that
already contains the four executables plus Qt DLLs/plugins (for example the
output of `clients/cpp/scripts/deploy.ps1` plus the Go binaries):

```powershell
.\packaging\msix\build-msix.ps1 `
  -PackageDir build\cpp\dist `
  -Version 0.5.0 `
  -OutFile build\weftlink-0.5.0-x64.msix
```

Add `-CertificatePfx cert.pfx -CertificatePassword <password>` to sign. The
manifest Publisher must match the certificate subject; pass `-Publisher` when
using a certificate other than `CN=Weftlink`.

## CI/CD signing

- **CI** generates a throwaway self-signed certificate (`CN=Weftlink`) and
  uploads the `.msix` plus its `.cer` as the `weftlink-msix` artifact.
- **CD** uses `MSIX_PFX_BASE64` + `MSIX_PFX_PASSWORD` repository secrets when
  present (recommended: a real code-signing certificate). Otherwise it falls
  back to the dev certificate and attaches its `.cer`.

## Installing a dev-signed MSIX

The certificate must be trusted before Windows will install the package:

```powershell
Import-Certificate -FilePath weftlink-signing.cer `
  -CertStoreLocation Cert:\LocalMachine\TrustedPeople
```

Then double-click the `.msix`, or:

```powershell
Add-AppxPackage .\weftlink-0.5.0-x64.msix
```

The app data (node key, TLS cert, knots, GUI log) stays in
`%USERPROFILE%\.weftlink`. The GUI's **Firewall rules** button requests
elevation and adds inbound TCP 7801 / UDP 7802 rules for the packaged daemon
path.
