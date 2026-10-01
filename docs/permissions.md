# Platform Permissions

> Central inventory of platform-specific permissions required by Weftlink.
> Update when adding new platform capabilities.

## macOS

| Feature        | Permission                            | Notes                |
|----------------|---------------------------------------|----------------------|
| Screen capture | Accessibility / Screen Recording       | System Preferences  |
| Input inject   | Accessibility / Assistive Tech        | Security & Privacy  |
| Clipboard      | (none — system clipboard APIs)        | —                    |
| Autostart      | Login items (LaunchAgent)             | —                    |

## Windows

| Feature        | Permission                            | Notes                |
|----------------|---------------------------------------|----------------------|
| Screen capture | (none — native APIs)                  | —                    |
| Input inject   | Admin or Assistive Tech flag          | robotgo / custom FFI |
| Clipboard      | (none — system APIs)                  | —                    |
| Autostart      | Run key / Startup folder              | —                    |

## Linux

| Feature        | Permission                            | Notes                |
|----------------|---------------------------------------|----------------------|
| Screen capture | (none — Wayland portal / X11)         | —                    |
| Input inject   | uinput group / $DISPLAY               | Wayland portal       |
| Clipboard      | (none — system APIs)                  | —                    |
| Autostart      | systemd user unit / .desktop          | —                    |

## Android

| Feature        | Permission                            | Notes                |
|----------------|---------------------------------------|----------------------|
| Screen capture | (TBD)                                 | —                    |
| Input inject   | Accessibility / Input Monitoring      | —                    |
| Clipboard      | Clipboard API                         | —                    |
| Autostart      | Background service intent             | —                    |

## HarmonyOS NEXT

| Feature        | Permission                            | Notes                |
|----------------|---------------------------------------|----------------------|
| Screen capture | ohos.permission.CAPTURE_SCREEN        | —                    |
| Input inject   | ohos.permission.INPUT_MONITORING      | —                    |
| Clipboard      | ohos.permission.ACCESS_CLIPBOARD      | —                    |
| Autostart      | ohos.permission.RUN_ON_BOOT           | —                    |