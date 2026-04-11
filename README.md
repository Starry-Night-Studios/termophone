# Termophone

Peer-to-peer voice calls in the terminal.

![Termophone](https://i.postimg.cc/0Q7CgCyC/Termophone.png)

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
brew install go pkg-config opus speexdsp rnnoise ffmpeg mpv
```

Linux (Debian/Ubuntu):

```bash
sudo apt update
sudo apt install -y golang pkg-config libopus-dev libspeexdsp-dev librnnoise-dev ffmpeg mpv
```

## Run

```bash
go mod download
go run .
```

First run creates:

- `~/.termophone/config.json`
- `~/.termophone/identity.key`
- `~/.termophone/store/`

## Controls

- `q`/`Ctrl+C`: quit
- `up`/`k`, `down`/`j`: move selection
- `Enter`/`Space`: call selected peer
- `y`/`Enter`: accept incoming call
- `n`/`Esc`: decline incoming call
- `m`: mute/unmute (in call)
- `v`: start/stop screen share (in call)
- `s`: settings (browsing), save peer (post-call)
- `r`: clear discovered peers list
- `x`/`delete`/`backspace`: remove saved contact
- `d`: toggle debug panel

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
  "screen_quality": "medium"
}
```

- `username`: shown to peers as `termophone/<username>`
- `contacts`: saved peer list
- `aec_trim_offset_ms`: AEC delay trim in ms
- `color_scheme`: UI theme index
- `screen_quality`: `low` | `medium` | `high`

## Test

```bash
go test . ./audio ./config ./net ./ui
```

## Status

- [x] Opus encoding
- [x] SpeexDSP AEC
- [x] RNNoise
- [x] Ring buffer overwrite handling
- [ ] PCM mixing with clamping
- [x] Ring buffer drift fix
- [x] libp2p
- [x] mDNS
- [x] DHT

## License

See `LICENSE`.
