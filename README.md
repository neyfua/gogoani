# Important, please read before createing Issues
- This is a fully vibe coded TUI tool built with [Go](https://go.dev/) for watching Anime directly on your terminal with [mpv](https://mpv.io/) as default media player
- This is a project for personal use so I **will not** accept any PRs, but Issues and bug reports are welcomed
- Do I ashamed of myself for using AI slops? **Fuck no**, I do ts because that fuckass crunchyroll company burned down some website's servers so I can't watch anime on those websites now, fuck you dogshitroll your services sucks
- Am I goated? Yes
- Why am I not accepting PRs? Cause ts is for personal use and I share it because I feel like it, I'm not advertising it or need people to donate or contribute codes
- You are **not** obligated to use `gogoani`, you can use others such as [ani-cli](https://github.com/pystardust/ani-cli), [GoAnime](https://github.com/alvarorichard/GoAnime) or any other TUI tools, and as I said this is just a project for personal use so if you plan on creating an Issue just to shame me then sybau ✌️
- Is LeBron goated? Yes
- 67
- Knicks or Spurs are gonna win the 2026 NBA Playoffs? I love Wemby but... KNICKS IN 4!!

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
- **mpv** - Media player (or set `GOGOANI_PLAYER` env var for alternatives)
- **fzf** (optional) - For fuzzy search interface
- **AniList API token** (optional) - For watch history sync

### Install gogoani

```bash
go install github.com/neyfua/gogoani/cmd/gogoani@latest
```

The `gogoani` binary will be installed to `~/.local/bin` (or `$GOPATH/bin` if set).

Make sure `~/.local/bin` or `$GOPATH/bin` is in your `PATH`:
```bash
# ~/.local/bin
export PATH="$HOME/.local/bin:$PATH"
```
```bash
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

### AniList Commands

```bash
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

Set a custom media player via environment variable:

```bash
export GOGOANI_PLAYER=vlc
```
