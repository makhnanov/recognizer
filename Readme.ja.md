![vibe](https://img.shields.io/badge/vibe-clauded-da6a46)

# Recognizer

**Linux用音声テキスト入力ツール** — キーを押しながら話し、離すとカーソル位置にテキストが表示されます。

[Русский](Readme.ru.md) | [English](Readme.md) | [Қазақша](Readme.kk.md) | [Українська](Readme.uk.md) | [日本語](Readme.ja.md) | [中文](Readme.zh.md) | [العربية](Readme.ar.md)

## 機能

- **プッシュトゥトーク録音** — F9を押して録音、離して文字起こし
- **即時テキスト挿入** — 文字起こしされたテキストは自動的にカーソル位置に貼り付けられます
- **クリップボード保持** — クリップボードの内容は保存され、貼り付け後に復元されます
- **音声フィードバック** — 録音開始/終了時にシロフォン音
- **ポインターフレーズ置換** — 「вот это」と言うとクリップボードの内容が挿入されます

## 要件

- X11を搭載したLinux
- Go 1.21以上
- PortAudioライブラリ
- OpenAI APIキー

## インストール

### 1. 依存関係をインストール

```bash
# Ubuntu/Debian
sudo apt install portaudio19-dev libx11-dev libxtst-dev libxinerama-dev libxcursor-dev libxkbcommon-dev

# Fedora
sudo dnf install portaudio-devel libX11-devel libXtst-devel libXinerama-devel libXcursor-devel libxkbcommon-devel

# Arch
sudo pacman -S portaudio libx11 libxtst libxinerama libxcursor libxkbcommon
```

### 2. クローンしてビルド

```bash
git clone https://github.com/makhnanov/recognizer.git
cd recognizer
cp .env.example .env
```

### 3. 設定

`.env`を編集してOpenAI APIキーを追加:

```
OPENAI_API_KEY=sk-...
```

### 4. ビルド

```bash
make
```

または手動で:

```bash
go build -o recognizer recognizer.go
```

## 使用方法

```bash
./recognizer
```

- **F9を押す** — 録音開始（ビープ音）
- **F9を離す** — 録音停止、文字起こし、貼り付け（ビープ音）
- **Ctrl+C** — 終了

### ポインターフレーズ

「вот это」、「вот эта」、「вот сюда」などのフレーズを言うと、現在のクリップボードの内容に置き換えられます。これにより、ディクテーション中にコピーしたテキストを参照できます。

例: コードをコピーし、「Исправь вот это」と言う → 「Исправь [クリップボードの内容]」が貼り付けられます

## シェルエイリアス

`~/.bashrc`または`~/.zshrc`に追加:

```bash
alias rec='/path/to/recognizer'
```

## 仕組み

1. システムフックを通じてF9キーイベントを監視
2. デフォルトマイクロフォンからWAVファイルに音声を録音
3. 音声をOpenAI Whisper APIに送信して文字起こし
4. テキストを処理（末尾の省略記号を削除、ポインターフレーズを置換）
5. Ctrl+Vで結果を貼り付け、元のクリップボードを保持

## ライセンス

MIT
