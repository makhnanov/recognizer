# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added
- Multilingual README support (English, Russian, Ukrainian, Kazakh, Japanese, Chinese, Arabic)
- Configurable transcription language via `WHISPER_LANGUAGE` environment variable
- LICENSE file (MIT)
- CHANGELOG.md

### Changed
- Extracted magic numbers into named constants for better maintainability
- Renamed Go module from `keywatch` to `recognizer`

## [1.0.0] - 2024

### Added
- Push-to-talk voice recording with F9 key
- OpenAI Whisper API integration for speech-to-text
- Automatic text insertion at cursor position via clipboard
- Clipboard content preservation and restoration
- Audio feedback (xylophone beep) on recording start/stop
- Pointer phrase substitution ("вот это", "вот эта", etc.) with clipboard content
- Trailing ellipsis removal from transcriptions
- Graceful shutdown on Ctrl+C
