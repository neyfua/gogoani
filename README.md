# Important, please read before createing Issues
- This is a fully vibe coded TUI tool built with [Go](https://go.dev/) for watching Anime directly on your terminal with [mpv](https://mpv.io/) as default media player
- This is a project for personal use so I **will not** accept any PRs, but Issues and bug reports are welcomed
- Am I goated? Yes
- 67

## Features

- Support for **subbed and dubbed** releases
- Fast **fzf-powered fuzzy search**
- **"Watch Another Anime"** menu for seamless anime switching
- **AniList support** (requires an AniList token):
    + Synchronize your AniList collection
    + Keep episode progress synced automatically
    + Check your anime list statuses directly
    + Modify your anime progress and status directly, with changes automatically synced to AniList

## Installation

### Prerequisites

- **Go 1.26+** - [Download](https://go.dev/dl/)
- **mpv** - Default media player (configure alternatives in `~/.config/gogoani/config.yaml`)
- **fzf** (optional) - For fuzzy search interface
- **AniList API token** (optional) - For watch history sync

### Install gogoani

```sh
go install github.com/neyfua/gogoani/cmd/gogoani@latest
```

The `gogoani` binary will be installed to `~/.local/bin` (or `$GOPATH/bin` if set).

Make sure `~/.local/bin` or `$GOPATH/bin` is in your `PATH`:
```sh
# ~/.local/bin
export PATH="$HOME/.local/bin:$PATH"
```
```sh
# $GOPATH/bin
export GOPATH=$HOME/go
export PATH="$GOPATH/bin:$PATH"
```

### Get AniList API Token (Optional)

1. Go to https://anilist.co/settings/developer
2. Click **"Create New Client"**
3. Fill in the form:
   - **Name**: Any name you prefer (or use gogoani)
   - **Redirect URL**: `https://anilist.co/api/v2/oauth/pin`
4. After creating, run `gogoani anilist --auth` in your terminal and paste the Client ID, Client Secret when prompted
5. Visit the link prompted out, copy the token and paste it into your terminal (the token will be stored in `~/.cache/gogoani/anilist_token.json` with secure permissions).

## Usage

### Basic Commands

```sh
# Interactive search
gogoani

# Direct search
gogoani "Overflow"

# Dubbed version
gogoani --dub "Redo of Healer"

# With debug logging
gogoani --debug "SHIROHIYO - Reincarnated as a Neglected Noble: Raising My Baby Brother With Memories From My Past Life"
```

### AniList Commands

```sh
# Authenticate with AniList
gogoani anilist --auth

# Sync your AniList anime list to local cache
gogoani anilist --sync

# View anime by status (watching, completed, paused, dropped)
gogoani anilist --status
gogoani anilist --status watching
gogoani anilist --status completed
gogoani anilist --status paused
gogoani anilist --status dropped
```

## Configuration

### Media Player

Set a custom media player in `~/.config/gogoani/config.yaml` (create the file manually):

```yaml
player: vlc # or mpv
```

Subtitle selection priority: the release's default subtitle, English, then the first available subtitle. Desktop mpv and VLC load external subtitles automatically.

On Android, mpv needs shared-storage permission. Add this line to `mpv.conf`:

```conf
include=/storage/emulated/0/mpv/mpv.config.mp4
```

Android VLC does not support referrers or external subtitles through its launch intent; use `android_mpv` instead.
