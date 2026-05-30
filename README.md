# gogoani

CLI tool to search and stream anime with episode navigation.

## Features

- Search anime by name
- Select from multiple search results
- Episode navigation menu (Next/Previous/Replay/Select/Quit)
- Dubbed and subbed versions support
- Non-blocking playback with immediate menu display
- Episode progress tracking

## Installation

### Prerequisites

- **Go 1.25+** - [Download](https://go.dev/dl/)
- **mpv** - Media player (or set `GOGOANI_PLAYER` env var for alternatives)
- **fzf** (optional) - For fuzzy search interface

#### Install dependencies:

**Arch Linux:**
```bash
sudo pacman -S go mpv fzf
```

**Ubuntu/Debian:**
```bash
sudo apt install golang mpv fzf
```

**macOS:**
```bash
brew install go mpv fzf
```

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
gogoani "anime name"

# Dubbed version
gogoani --dub "anime name"

# With debug logging
gogoani --debug "anime name"
```

### Episode Navigation

After selecting an episode, mpv will start playing and a menu will appear:

- **Next Episode** - Play next episode
- **Previous Episode** - Play previous episode
- **Select Different Episode** - Choose any episode
- **Replay Current Episode** - Restart current episode
- **Quit** - Exit the application

Press `Ctrl+C` anytime to quit.

## Configuration

Set a custom media player via environment variable:

```bash
export GOGOANI_PLAYER=vlc
```

## Contributing

**Note:** This is a personal project for my own use. I will **not accept Pull Requests**.

However, **Issues and bug reports are welcome!** If you encounter a bug or have a suggestion, please open an issue.

## License

This project is for personal use only.
