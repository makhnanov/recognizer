# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Recognizer is a Linux voice-to-text input tool written in Go. Hold F9, speak, release, and transcribed text is pasted at cursor position. It uses OpenAI Whisper API for speech recognition and PortAudio for audio capture.

## Build & Run Commands

- `make` — build and run (installs deps, compiles, launches)
- `make build` — compile binary only
- `make install` — run `go mod tidy`
- `make clean` — remove binary and recorded WAV files from `.voices/`
- `go build -o recognizer recognizer.go` — manual build

## Configuration

The app loads `.env` from the executable's directory (not CWD). Required variables:
- `OPENAI_API_KEY` — OpenAI API key (required)
- `WHISPER_LANGUAGE` — language hint for Whisper (optional, e.g. `ru`)

Copy `.env.example` to `.env` to get started.

## System Dependencies

Requires native libraries (must be installed before building):
```
sudo apt install portaudio19-dev libx11-dev libxtst-dev libxinerama-dev libxcursor-dev libxkbcommon-dev
```

## Architecture

Single-file Go application (`recognizer.go`) with no internal packages. All logic is in one file:

- **AudioRecorder struct** — manages PortAudio stream, writes PCM data to WAV files in `.voices/` directory. Uses goroutine with stop/done channels for recording loop.
- **transcribeAudio()** — sends WAV file to OpenAI Whisper API via multipart POST, returns transcribed text.
- **copyToClipboardAndPaste()** — saves current clipboard, writes text to clipboard, simulates Ctrl+V, then restores original clipboard.
- **replacePointerPhrases()** — substitutes Russian pointer phrases ("вот это", etc.) with current clipboard content. Checks case-insensitive, replaces in original/capitalized/uppercase forms.
- **playBeep()** — generates a synthesized xylophone sound (C6 note with harmonics and exponential decay) via PortAudio.
- **main()** — loads `.env`, initializes PortAudio, listens for F9 key events via `gohook`, orchestrates record→transcribe→paste flow.

## Key Dependencies

- `github.com/gordonklaus/portaudio` — audio I/O (record + beep playback)
- `github.com/robotn/gohook` — global keyboard hook (F9 key detection)
- `github.com/go-vgo/robotgo` — clipboard access and key simulation (Ctrl+V paste)
- `github.com/joho/godotenv` — `.env` file loading

## Important Constants

F9 key rawcode is `65478`. Audio: 44100 Hz, 16-bit mono. API timeout: 30 seconds.

## Notes

- X11 only (no Wayland support)
- `autostart/recognizer.desktop` is an XFCE autostart entry
- WAV files are stored in `.voices/` next to the binary (created at runtime)
- `.env` is loaded relative to executable path, not working directory
