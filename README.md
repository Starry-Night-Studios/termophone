# Termophone

Peer-to-peer voice calls in the terminal!

![Termophone](https://github.com/kurobyte19/U1907-site-images/blob/main/projects/screenshot.png?raw=true)

## Requirements

- Go 1.25+
- pkg-config
- opus
- speexdsp
- rnnoise
- ffmpeg (for screen share)
- mpv (for screen share receive)

macOS (Homebrew):

```bash
brew install go pkg-config opus speexdsp ffmpeg mpv
```

Linux (Debian/Ubuntu):

```bash
sudo apt update
sudo apt install -y golang pkg-config libopus-dev libspeexdsp-dev ffmpeg mpv
```

> **Note:** `rnnoise` must be built from source. Please refer to the [xiph/rnnoise repository](https://github.com/xiph/rnnoise) or their official documentation for build instructions.

## Run

```bash
go mod download
go run .
```

First run creates:

- `~/.termophone/config.json`
- `~/.termophone/identity.key`

## Global Setup

For global discovery and connectivity across different networks, you will need to set up the following infrastructure:

- [Lobby Server](https://github.com/kurobyte19/lobby)
- [Relay Server](https://github.com/kurobyte19/relay)

## Controls

- `q`/`Ctrl+C`: quit
- `tab`: switch focus between panes
- `up`/`k`, `down`/`j`: move selection
- `Enter`/`Space`: call selected peer
- `c`: connect to lobby server
- `l`: disconnect from lobby server
- `y`/`Enter`: accept incoming call
- `n`/`Esc`: decline incoming call
- `m`: mute/unmute (in call)
- `v`: start/stop screen share (in call)
- `s`: settings (browsing), save peer (post-call)
- `r`: clear discovered peers list
- `x`/`delete`/`backspace`: remove saved contact
- `d`: toggle debug panel

**In Settings:**
- `up`/`down`: navigate settings
- `left`/`right`: change setting value
- `Enter`: save and exit settings
- `Esc`: discard and exit settings

## Config

Path: `~/.termophone/config.json`

```json
{
  "username": "Anon",
  "contacts": [
    { "name": "Alice", "peer_id": "12D3KooW..." }
  ],
  "aec_trim_offset_ms": 120,
  "color_scheme": 0,
  "screen_quality": "medium",
  "lobby_server": "ws://lobby.example.com/ws"
}
```

- `username`: shown to peers as `termophone/<username>`
- `contacts`: saved peer list
- `aec_trim_offset_ms`: AEC delay trim in ms
- `color_scheme`: UI theme index
- `screen_quality`: `low` | `medium` | `high`
- `lobby_server`: WebSocket URL of the global discovery lobby

## Test

```bash
go test . ./audio ./config ./net ./ui
```

## Status

- [x] Opus encoding
- [x] SpeexDSP AEC
- [x] RNNoise
- [x] Ring buffer overwrite handling
- [x] Ring buffer drift fix
- [x] libp2p
- [x] mDNS
- [x] DHT
- [ ] PCM mixing with clamping

## License

See `LICENSE`.
