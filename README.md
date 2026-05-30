# Important, please read before createing Issues
- This is a fully vibe coded TUI tool built with [Go](https://go.dev/) for watching Anime directly on your terminal with [mpv](https://mpv.io/) as default media player
- This is a project for personal use so I **will not** accept any PRs, but Issues and bug reports are welcomed
- Do I ashamed of myself for using AI slops? **Fuck no**, I do ts because that fuckass crunchyroll company burned down some website's servers so I can't watch anime on those websites now, fuck you dogshitroll your services sucks
- Am I goated? Yes
- Why am I not accepting PRs? Cause ts is for personal use and I share it because I feel like it, I'm not advertising it or need people to donate or contribute codes
- You are **not** obligated to use `gogoani`, you can use others such as [ani-cli](https://github.com/pystardust/ani-cli), [GoAnime](https://github.com/alvarorichard/GoAnime) or any other TUI tools, and as I said this is just a project for personal use so if you plan on creating an Issue just to shame me then sybau ✌️
- Is LeBron goated? Yes
- 67

## Features

- Search anime by name
- Dubbed and subbed versions support
- Episode progress tracking

## Installation

### Prerequisites

- **Go 1.25+** - [Download](https://go.dev/dl/)
- **mpv** - Media player (or set `GOGOANI_PLAYER` env var for alternatives)
- **fzf** (optional) - For fuzzy search interface

### Install gogoani

```bash
go install github.com/neyfua/gogoani/cmd/gogoani@latest
```

The `gogoani` binary will be installed to `~/.local/bin` (or `$GOPATH/bin` if set).

Make sure `~/.local/bin` is in your `PATH`:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Usage

```bash
# Interactive search
gogoani

# Direct search
gogoani "Overflow"

# Dubbed version
gogoani --dub "Redo of Healer"

# With debug logging
gogoani --debug "Boku no Pico"
```

## Configuration

Set a custom media player via environment variable:

```bash
export GOGOANI_PLAYER=vlc
```

## License

This project is for personal use only.
