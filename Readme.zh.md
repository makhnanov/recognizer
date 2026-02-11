![vibe](https://img.shields.io/badge/vibe-clauded-da6a46)

# Recognizer

**Linux语音转文字输入工具** — 按住按键，说话，松开，文字就会出现在光标位置。

[Русский](Readme.ru.md) | [English](Readme.md) | [Қазақша](Readme.kk.md) | [Українська](Readme.uk.md) | [日本語](Readme.ja.md) | [العربية](Readme.ar.md)

## 功能特点

- **按键说话录音** — 按住F9录音，松开进行转录
- **即时文本插入** — 转录的文本自动粘贴到光标位置
- **智能粘贴** — 在终端中自动使用Ctrl+Shift+V，在其他应用中使用Ctrl+V
- **剪贴板保留** — 剪贴板内容在粘贴后会保存并恢复
- **音频反馈** — 录音开始/结束时有木琴提示音
- **指示短语替换** — 说"这个"可插入剪贴板内容
- **多语言短语** — 支持俄语、英语、乌克兰语、哈萨克语、日语、中文和阿拉伯语
- **可配置模型** — 通过`.env`选择任意OpenAI转录模型

## 系统要求

- 带X11的Linux
- Go 1.21+
- PortAudio库
- OpenAI API密钥

## 安装

### 1. 安装依赖

```bash
# Ubuntu/Debian
sudo apt install portaudio19-dev libx11-dev libxtst-dev libxinerama-dev libxcursor-dev libxkbcommon-dev

# Fedora
sudo dnf install portaudio-devel libX11-devel libXtst-devel libXinerama-devel libXcursor-devel libxkbcommon-devel

# Arch
sudo pacman -S portaudio libx11 libxtst libxinerama libxcursor libxkbcommon
```

### 2. 克隆并构建

```bash
git clone https://github.com/makhnanov/recognizer.git
cd recognizer
cp .env.example .env
```

### 3. 配置

编辑`.env`并设置参数：

```
OPENAI_API_KEY=sk-...
WHISPER_MODEL=gpt-4o-transcribe
WHISPER_LANGUAGE=zh
```

- `OPENAI_API_KEY` — 您的OpenAI API密钥（必需）
- `WHISPER_MODEL` — 转录模型（默认：`gpt-4o-transcribe`）
- `WHISPER_LANGUAGE` — ISO-639-1格式的语言提示（可选，例如`zh`、`en`）

### 4. 构建

```bash
make
```

或手动构建：

```bash
go build -o recognizer recognizer.go
```

## 使用方法

```bash
./recognizer
```

- **按住F9** — 开始录音（提示音）
- **松开F9** — 停止录音、转录并粘贴（提示音）
- **Ctrl+C** — 退出

### 指示短语

指示短语从`phrases.txt`文件加载（每行一个短语，`#`行为注释）。文件在每次转录时重新读取，因此可以在不重启程序的情况下编辑。

开箱即用支持多种语言的短语：俄语、英语、乌克兰语、哈萨克语、日语、中文和阿拉伯语。

示例：复制一些代码，然后说"修复这个" → 粘贴"修复[剪贴板内容]"

## 自动启动

要在登录时自动启动Recognizer，复制桌面入口文件：

```bash
cp scripts/recognizer.desktop $HOME/.config/scripts/
```

## 重新构建和重启

停止当前实例、重新构建并在后台启动：

```bash
./scripts/restart.sh
```

脚本会查找并停止正在运行的进程，构建新的二进制文件，并将其从终端分离启动。输出记录到`recognizer.log`。

## Shell别名

添加到`~/.bashrc`或`~/.zshrc`：

```bash
alias rec='/path/to/recognizer'
```

## 工作原理

1. 通过系统钩子监听F9键事件
2. 从默认麦克风录制音频到WAV文件
3. 将音频发送到OpenAI转录API（可配置模型）
4. 处理文本（删除末尾省略号，替换`phrases.txt`中的指示短语）
5. 检测活动窗口类型，通过Ctrl+Shift+V（终端）或Ctrl+V（其他应用）粘贴结果，保留原始剪贴板

## 许可证

MIT
