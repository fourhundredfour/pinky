<div align="center">

# 🌸 pinky

**Give your Windows 11 taskbar a colored makeover — without losing your icons.**

Turn every taskbar icon monochrome, tinted pink, or any color you like, while staying perfectly recognizable — like a color filter laid over your taskbar.

</div>

---

## ✨ What it does

pinky sits quietly on top of your taskbar and recolors it live, using the same kind of "blend mode + opacity" trick you'd use on a layer in Photoshop. Your icons keep their shapes and details — only their color changes.

- 🎨 Pick any color
- 🌗 Control the strength (opacity)
- 🪄 Choose how it blends: monochrome, tint, multiply, or color
- 🎯 Colors only the **icons and text**
- ⚡ Updates live, runs quietly in your system tray
- 🖥️ Works across all your monitors

## 📥 Installation

1. Go to the [Releases](../../releases) page and download the latest `pinky-windows-amd64.exe` (or `pinky-windows-arm64.exe` if you're on an ARM-based Windows PC).
2. Put it anywhere you like, e.g. `C:\Tools\pinky\`.
3. Double-click it to run. A tray icon will appear, and your taskbar will start changing color.

> **Note:** pinky needs Windows 11 (or Windows 10 version 2004+).

## 🚀 Getting started

The first time pinky runs, it creates a `config.yaml` file next to the `.exe` with defaults — a pink monochrome look at 80% strength. Open that file in any text editor to customize it, save, and pinky will pick up your changes automatically within a second — no restart needed.

```yaml
enabled: true          # turn the effect on/off
color: "#FF33AA"       # pick any color, as #RRGGBB
opacity: 0.8           # how strong the effect is, from 0.0 to 1.0
mode: monochrome       # monochrome | tint | multiply | color
fps: 30                # how smoothly it updates
include_tray: true     # also color the clock/system tray icons?
icon_sensitivity: 0.5  # how much of each icon gets colored (0.0–1.0)
```

> pinky colors just the icon and text pixels, leaving the space between them untouched — so a transparent or "hidden" taskbar stays transparent. If some faint icon detail isn't getting colored, nudge `icon_sensitivity` up; if you see stray specks, nudge it down.

## 🎨 Blend modes

| Mode | Look |
|---|---|
| 🖤 `monochrome` | Classic effect — icons turn grayscale, then get tinted with your color. Great for an "all-pink taskbar" look. |
| 🖌️ `tint` | A flat colored film over everything, like a stained-glass overlay. |
| ✖️ `multiply` | Darkens icons using your color — moodier, more saturated shadows. |
| 🌈 `color` | Keeps each icon's brightness, just swaps its hue — the most "colorized photo" look. |

## 🧰 Using the tray icon

Click the pinky icon in your system tray to:

- **Enable / Disable** — instantly toggle the effect
- **Reload config** — force-apply changes you just saved
- **Quit pinky** — close the app

## ❓ FAQ

**Does this slow down my PC?**
No — it only redraws a small strip of the screen, a handful of times per second. CPU usage is minimal.

**Can I still click my taskbar icons normally?**
Yes! The color layer never blocks clicks, hovering, or right-click menus.

**Only part of my taskbar gets colored / an icon is missing its color.**
pinky finds icons using Windows' accessibility (UI Automation) data, which refreshes a few times a second. A brand-new icon might stay uncolored for a fraction of a second. If a whole category is never colored, it may be a newer Windows layout, please [open an issue](../../issues).

**Will this break with a Windows update?**
It's built on standard, documented Windows APIs, but if a future Windows update changes how the taskbar looks, please [open an issue](../../issues). Even if icon detection changes, pinky automatically falls back to a best-effort mode so the effect keeps working.
