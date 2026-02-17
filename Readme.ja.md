![vibe](https://img.shields.io/badge/vibe-clauded-da6a46)

# Recognizer

**Linux用音声テキスト入力ツール** — キーを押しながら話し、離すとカーソル位置にテキストが表示されます。

[Русский](Readme.ru.md) | [English](Readme.md) | [Қазақша](Readme.kk.md) | [Українська](Readme.uk.md) | [中文](Readme.zh.md) | [العربية](Readme.ar.md)

## 機能

- **プッシュトゥトーク録音** — F9を押して録音、離して文字起こし
- **即時テキスト挿入** — 文字起こしされたテキストは自動的にカーソル位置に貼り付けられます
- **スマート貼り付け** — ターミナルではCtrl+Shift+V、その他のアプリではCtrl+Vを自動使用
- **クリップボード保持** — クリップボードの内容は保存され、貼り付け後に復元されます
- **音声フィードバック** — 録音開始/終了時にシロフォン音
- **ポインターフレーズ置換** — 「これ」と言うとクリップボードの内容が挿入されます
- **多言語フレーズ** — ロシア語、英語、ウクライナ語、カザフ語、日本語、中国語、アラビア語対応
- **設定可能なモデル** — `.env`で任意のOpenAI文字起こしモデルを選択

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

`.env`を編集してパラメータを設定:

```
OPENAI_API_KEY=sk-...
WHISPER_MODEL=gpt-4o-transcribe
WHISPER_LANGUAGE=ja
```

- `OPENAI_API_KEY` — OpenAI APIキー（必須）
- `WHISPER_MODEL` — 文字起こしモデル（デフォルト: `gpt-4o-transcribe`）
- `WHISPER_LANGUAGE` — ISO-639-1形式の言語ヒント（任意、例: `ja`、`en`）

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

ポインターフレーズは`phrases.txt`ファイルから読み込まれます（1行に1フレーズ、`#`行はコメント）。ファイルは文字起こしのたびに再読み込みされるため、プログラムを再起動せずに編集できます。

複数言語のフレーズが標準搭載: ロシア語、英語、ウクライナ語、カザフ語、日本語、中国語、アラビア語。

例: コードをコピーし、「これを直して」と言う → 「[クリップボードの内容]を直して」が貼り付けられます

## 自動起動

ログイン時にRecognizerを自動起動するには、デスクトップエントリをコピー:

```bash
cp scripts/recognizer.desktop $HOME/.config/autostart/ # If executor in /var/www/recognizer/recognizer
```

## リビルドと再起動

実行中のインスタンスを停止し、リビルドしてバックグラウンドで起動:

```bash
./scripts/restart.sh
```

スクリプトは実行中のプロセスを見つけて停止し、新しいバイナリをビルドし、ターミナルから切り離して起動します。出力は`recognizer.log`に記録されます。

## シェルエイリアス

`~/.bashrc`または`~/.zshrc`に追加:

```bash
alias rec='/path/to/recognizer'
```

## 仕組み

1. システムフックを通じてF9キーイベントを監視
2. デフォルトマイクロフォンからWAVファイルに音声を録音
3. 音声をOpenAI文字起こしAPIに送信（設定可能なモデル）
4. テキストを処理（末尾の省略記号を削除、`phrases.txt`からポインターフレーズを置換）
5. アクティブウィンドウの種類を検出し、Ctrl+Shift+V（ターミナル）またはCtrl+V（その他）で結果を貼り付け、元のクリップボードを保持

## ライセンス

MIT
