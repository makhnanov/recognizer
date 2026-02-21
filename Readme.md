![vibe](https://img.shields.io/badge/vibe-clauded-da6a46)

# Recognizer

**Voice-to-text input tool for Linux** — hold a key, speak, release, and your words appear at the cursor.

[Русский](Readme.ru.md) | [Қазақша](Readme.kk.md) | [Українська](Readme.uk.md) | [日本語](Readme.ja.md) | [中文](Readme.zh.md) | [العربية](Readme.ar.md)

## Features

- **Push-to-talk recording** — hold F9 to record, release to transcribe
- **Instant text insertion** — transcribed text is automatically pasted at cursor position
- **Smart paste** — automatically uses Ctrl+Shift+V in terminal emulators, Ctrl+V elsewhere
- **Clipboard preservation** — your clipboard content is saved and restored after paste
- **Audio feedback** — xylophone beep on start/stop recording
- **Pointer phrase substitution** — say "this thing" or "вот это" to insert clipboard content inline
- **Multi-language phrases** — pointer phrases supported in Russian, English, Ukrainian, Kazakh, Japanese, Chinese, and Arabic
- **Negativity filter** — automatically rewrites rude or offensive speech in a polite way (configurable threshold)
- **Configurable transcription model** — choose any OpenAI transcription model via `.env`

## Requirements

- Linux with X11
- Go 1.21+
- PortAudio library
- OpenAI API key

## Installation

### 1. Install dependencies

```bash
# Ubuntu/Debian
sudo apt install portaudio19-dev libx11-dev libxtst-dev libxinerama-dev libxcursor-dev libxkbcommon-dev

# Fedora
sudo dnf install portaudio-devel libX11-devel libXtst-devel libXinerama-devel libXcursor-devel libxkbcommon-devel

# Arch
sudo pacman -S portaudio libx11 libxtst libxinerama libxcursor libxkbcommon
```

### 2. Clone and build

```bash
git clone https://github.com/makhnanov/recognizer.git
cd recognizer
cp .env.example .env
```

### 3. Configure

Edit `.env` and set your parameters:

```
OPENAI_API_KEY=sk-...
WHISPER_MODEL=gpt-4o-transcribe
WHISPER_LANGUAGE=ru
FILTER_MODEL=gpt-4o-mini
NEGATIVITY_THRESHOLD=50
```

- `OPENAI_API_KEY` — your OpenAI API key (required)
- `WHISPER_MODEL` — transcription model (default: `gpt-4o-transcribe`)
- `WHISPER_LANGUAGE` — language hint in ISO-639-1 format (optional, e.g. `ru`, `en`)
- `FILTER_MODEL` — model for negativity analysis (default: `gpt-4o-mini`)
- `NEGATIVITY_THRESHOLD` — negativity percentage threshold for rewriting (default: `50`, set to `0` to disable)

### 4. Build

```bash
make
```

Or manually:

```bash
go build -o recognizer recognizer.go
```

## Usage

```bash
./recognizer
```

- **Hold F9** — start recording (beep sound)
- **Release F9** — stop recording, transcribe, and paste (beep sound)
- **Ctrl+C** — exit

### Pointer Phrases

Pointer phrases are loaded from the `phrases.txt` file (one phrase per line, `#` lines are comments). The file is re-read on every transcription, so you can edit it without restarting the program.

Multi-language phrases are included out of the box: Russian, English, Ukrainian, Kazakh, Japanese, Chinese, and Arabic.

Example: Copy some code, then say "Fix this thing" → pastes "Fix [your clipboard content]"

## Autostart

To start Recognizer automatically on login, copy the desktop entry:

```bash
cp scripts/recognizer.desktop $HOME/.config/autostart/ # If executor in /var/www/recognizer/recognizer
```

## Rebuild and Restart

To stop the running instance, rebuild, and launch in the background:

```bash
./scripts/restart.sh
```

The script finds and stops the running process, builds a fresh binary, and starts it detached from the terminal. Output is logged to `recognizer.log`.

## Shell Aliases

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
alias rec='/path/to/recognizer'
```

## How It Works

1. Listens for F9 key events via system hooks
2. Records audio from default microphone to WAV file
3. Sends audio to OpenAI transcription API (configurable model)
4. Processes text (removes trailing ellipsis, substitutes pointer phrases from `phrases.txt`)
5. Analyzes text negativity — if it exceeds the threshold, rewrites it politely
6. Detects active window type and pastes result via Ctrl+Shift+V (terminals) or Ctrl+V (other apps), preserving original clipboard

## License

MIT
