<div align="center">

# noct

*minimal. fast. beautifully terminal.*

[![Go](https://img.shields.io/badge/Go-1.27-00ADD8.svg?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Bubble Tea](https://img.shields.io/badge/Bubble%20Tea-TUI-FF75B7.svg?style=flat-square)](https://github.com/charmbracelet/bubbletea)
[![Lip Gloss](https://img.shields.io/badge/Lip%20Gloss-Styles-C084FC.svg?style=flat-square)](https://github.com/charmbracelet/lipgloss)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-22D3EE.svg?style=flat-square)](LICENSE)

</div>

## Features

- **Instant search** — Find and queue tracks in seconds
- **Clean TUI** — Focused, distraction-free player built for the terminal
- **Smart queue** — Effortlessly build and navigate your playlist
- **Now playing** — Sleek progress bar with time and playback state
- **Keyboard-first** — Fluid, vim-inspired controls

## Requirements

- [`mpv`](https://mpv.io/) — Audio playback
- [`yt-dlp`](https://github.com/yt-dlp/yt-dlp) — Stream extraction

Make sure both are available in your `$PATH`.

## Installation

### With `go install`

```bash
go install github.com/PhotoDotPi/noct/cmd/noct@latest
```

### From source

```bash
git clone https://github.com/PhotoDotPi/noct.git
cd noct
go build -o noct ./cmd/noct
./noct
```

## Keybinds

| Key | Action |
| :-- | :-- |
| `/` | Search for music |
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Play selected track |
| `Space` | Play / Pause |
| `q` / `Ctrl+C` | Quit |
| `?` | Toggle help |

## Built with

- [`bubbletea`](https://github.com/charmbracelet/bubbletea) — TUI framework
- [`bubbles`](https://github.com/charmbracelet/bubbles) — UI components
- [`lipgloss`](https://github.com/charmbracelet/lipgloss) — Terminal styling

## License

[AGPL-3.0](LICENSE) © PhotoDotPi
