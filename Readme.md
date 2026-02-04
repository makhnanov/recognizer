![vibe](https://img.shields.io/badge/vibe-clauded-da6a46)

# Recognize

**Voice-to-text input tool for Linux** — hold a key, speak, release, and your words appear at the cursor.

[Русский](Readme.ru.md) | [English](Readme.md) | [Қазақша](Readme.kk.md) | [Українська](Readme.uk.md) | [日本語](Readme.ja.md) | [中文](Readme.zh.md) | [العربية](Readme.ar.md)

## Features

- **Push-to-talk recording** — hold F9 to record, release to transcribe
- **Instant text insertion** — transcribed text is automatically pasted at cursor position
- **Clipboard preservation** — your clipboard content is saved and restored after paste
- **Audio feedback** — xylophone beep on start/stop recording
- **Pointer phrase substitution** — say "вот это" (this one) to insert clipboard content inline

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

Edit `.env` and add your OpenAI API key:

```
OPENAI_API_KEY=sk-...
```

### 4. Build

```bash
make
```

Or manually:

```bash
go build -o recognize recognize.go
```

## Usage

```bash
./recognize
```

- **Hold F9** — start recording (beep sound)
- **Release F9** — stop recording, transcribe, and paste (beep sound)
- **Ctrl+C** — exit

### Pointer Phrases

When you say phrases like "вот это", "вот эта", "вот сюда" etc., they will be replaced with your current clipboard content. This allows you to reference copied text in your dictation.

Example: Copy some code, then say "Исправь вот это" → pastes "Исправь [your clipboard content]"

## Shell Aliases

Add to your `~/.bashrc` or `~/.zshrc`:

```bash
alias rec='/path/to/recognize'
```

## How It Works

1. Listens for F9 key events via system hooks
2. Records audio from default microphone to WAV file
3. Sends audio to OpenAI Whisper API for transcription
4. Processes text (removes trailing ellipsis, substitutes pointer phrases)
5. Pastes result via Ctrl+V, preserving original clipboard

## License

MIT
