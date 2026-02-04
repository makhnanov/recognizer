![vibe](https://img.shields.io/badge/vibe-clauded-da6a46)

# Recognize

**Linux语音转文字输入工具** — 按住按键，说话，松开，文字就会出现在光标位置。

[Русский](Readme.ru.md) | [English](Readme.md) | [Қазақша](Readme.kk.md) | [Українська](Readme.uk.md) | [日本語](Readme.ja.md) | [中文](Readme.zh.md) | [العربية](Readme.ar.md)

## 功能特点

- **按键说话录音** — 按住F9录音，松开进行转录
- **即时文本插入** — 转录的文本自动粘贴到光标位置
- **剪贴板保留** — 剪贴板内容在粘贴后会保存并恢复
- **音频反馈** — 录音开始/结束时有木琴提示音
- **指示短语替换** — 说"вот это"可插入剪贴板内容

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

编辑`.env`并添加您的OpenAI API密钥：

```
OPENAI_API_KEY=sk-...
```

### 4. 构建

```bash
make
```

或手动构建：

```bash
go build -o recognize recognize.go
```

## 使用方法

```bash
./recognize
```

- **按住F9** — 开始录音（提示音）
- **松开F9** — 停止录音、转录并粘贴（提示音）
- **Ctrl+C** — 退出

### 指示短语

当您说"вот это"、"вот эта"、"вот сюда"等短语时，它们会被替换为当前剪贴板内容。这允许您在听写中引用复制的文本。

示例：复制一些代码，然后说"Исправь вот это" → 粘贴"Исправь [剪贴板内容]"

## Shell别名

添加到`~/.bashrc`或`~/.zshrc`：

```bash
alias rec='/path/to/recognize'
```

## 工作原理

1. 通过系统钩子监听F9键事件
2. 从默认麦克风录制音频到WAV文件
3. 将音频发送到OpenAI Whisper API进行转录
4. 处理文本（删除末尾省略号，替换指示短语）
5. 通过Ctrl+V粘贴结果，保留原始剪贴板

## 许可证

MIT
