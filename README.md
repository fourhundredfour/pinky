<div align="center">

# 🌸 pinky

**A fast, fully customizable Windows 11 taskbar replacement.**

Docks to any screen edge, shows your open windows, the clock, and system indicators (battery/network/volume) - transparent, rounded or square, left/center/right aligned, monochrome icons, all live-reloadable from a single TOML file.

</div>

---

## ✨ Features

- 🪟 **Open windows** - icons for every open app, click to focus, middle-click to close
- 🕒 **Clock** - configurable time/date format
- 🔋 **System indicators** - battery, network, volume, with click-through to Windows' own Quick Settings flyouts
- 🎨 **Fully customizable** - dock edge, thickness, alignment, transparency, square/rounded corners, monochrome icons
- ⚡ **Live-reloading TOML config** - edit `config.toml`, save, see it applied instantly - no restart

> **Scope:** primary monitor only, and only Windows' own system tray flyouts - no capturing of third-party tray icons yet.

## 📋 Requirements

- Windows 11 (or Windows 10 2004+)
- The [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/) - already preinstalled on Windows 11 and current Windows 10 builds.

## 📥 Installation

1. Go to the [Releases](../../releases) page and download the latest `pinky-windows-amd64.exe`.
2. Put it anywhere you like, e.g. `C:\Tools\pinky\`.
3. Double-click it to run. Windows' own taskbar hides, pinky's bar appears in its place, and a tray icon shows up for settings/reload/quit.

The first run creates `%AppData%\pinky\config.toml` (i.e. `C:\Users\<you>\AppData\Roaming\pinky\config.toml`) pre-filled with defaults. Edit it in any text editor and save - pinky picks up the change within a fraction of a second.

## 🔧 Configuration

See `config.example.toml` for the full, commented reference. Every field:

| Key | Type | Default | Description |
|---|---|---|---|
| `edge` | `"top"\|"bottom"\|"left"\|"right"` | `"bottom"` | Screen edge the bar docks to |
| `size` | int (px) | `48` | Bar thickness (height if top/bottom, width if left/right) |
| `alignment` | `"left"\|"center"\|"right"` | `"center"` | Where the open-window icons sit; clock/indicators always stay at the far end |
| `shape` | `"square"\|"rounded"` | `"rounded"` | Bar corner style (uses DWM rounded corners on Windows 11) |
| `background_color` | `"#RRGGBB"` | `"#101014"` | Bar background color |
| `background_opacity` | float `0.0`-`1.0` | `0.85` | Bar background strength; `0` is fully transparent |
| `accent_color` | `"#RRGGBB"` | `"#3AA0FF"` | Highlights the active window / hover states |
| `monochrome_icons` | bool | `false` | Desaturate running-app icons for a flatter look |
| `clock_format` | Go time layout | `"15:04"` | e.g. `"03:04 PM"` for 12h |
| `date_format` | Go time layout | `"Monday, 02 January 2006"` | Used for the clock's tooltip |
| `show_tasks` / `show_clock` / `show_battery` / `show_network` / `show_volume` | bool | `true` | Toggle individual widgets off without losing their settings |
| `hide_real_taskbar` | bool | `true` | Hide Explorer's own taskbar while pinky runs (restored on exit); turn off to run both side by side, e.g. while developing |
| `tasks_poll_interval_ms` | int | `1500` | Fallback poll interval for the open-window list (in addition to live shell-hook notifications) |
| `indicators_poll_interval_ms` | int | `2000` | Poll interval for battery/network/volume |
| `monitor` | `"primary"` | `"primary"` | Reserved for future multi-monitor support |

Use the tray icon's **"Open config.toml"** to jump straight to the file, and **"Reload config"** to force a re-read if your editor didn't trigger the file watcher.

## ❓ FAQ

**Does this replace Explorer entirely?**
No - only the taskbar window. Explorer itself, the Start menu, and desktop icons are untouched; pinky just hides the real taskbar and reserves its screen edge for itself.

**What happens if `explorer.exe` restarts or crashes?**
pinky listens for the systemwide `TaskbarCreated` broadcast and automatically re-hides the taskbar Explorer just recreated.

**Can I run it alongside the real taskbar?**
Yes - set `hide_real_taskbar = false` in the config.
