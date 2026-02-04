![vibe](https://img.shields.io/badge/vibe-clauded-da6a46)

# Recognize

**Голосове введення тексту для Linux** — затисніть клавішу, говоріть, відпустіть — і текст з'явиться в позиції курсора.

[Русский](Readme.ru.md) | [English](Readme.md) | [Қазақша](Readme.kk.md) | [Українська](Readme.uk.md) | [日本語](Readme.ja.md) | [中文](Readme.zh.md) | [العربية](Readme.ar.md)

## Можливості

- **Push-to-talk запис** — затисніть F9 для запису, відпустіть для розпізнавання
- **Миттєва вставка** — розпізнаний текст автоматично вставляється в позицію курсора
- **Збереження буфера обміну** — вміст буфера зберігається і відновлюється після вставки
- **Звуковий зворотний зв'язок** — звук ксилофона при початку/закінченні запису
- **Підстановка вказівних фраз** — скажіть "вот это" для вставки вмісту буфера обміну

## Вимоги

- Linux з X11
- Go 1.21+
- Бібліотека PortAudio
- API-ключ OpenAI

## Встановлення

### 1. Встановіть залежності

```bash
# Ubuntu/Debian
sudo apt install portaudio19-dev libx11-dev libxtst-dev libxinerama-dev libxcursor-dev libxkbcommon-dev

# Fedora
sudo dnf install portaudio-devel libX11-devel libXtst-devel libXinerama-devel libXcursor-devel libxkbcommon-devel

# Arch
sudo pacman -S portaudio libx11 libxtst libxinerama libxcursor libxkbcommon
```

### 2. Клонуйте та зберіть

```bash
git clone https://github.com/makhnanov/recognizer.git
cd recognizer
cp .env.example .env
```

### 3. Налаштуйте

Відредагуйте `.env` і додайте ваш API-ключ OpenAI:

```
OPENAI_API_KEY=sk-...
```

### 4. Зберіть

```bash
make
```

Або вручну:

```bash
go build -o recognize recognize.go
```

## Використання

```bash
./recognize
```

- **Затиснути F9** — почати запис (звуковий сигнал)
- **Відпустити F9** — зупинити запис, розпізнати і вставити (звуковий сигнал)
- **Ctrl+C** — вихід

### Вказівні фрази

Коли ви вимовляєте фрази на кшталт "вот это", "вот эта", "вот сюда" тощо, вони замінюються на поточний вміст буфера обміну. Це дозволяє посилатися на скопійований текст у диктуванні.

Приклад: Скопіюйте код, потім скажіть "Исправь вот это" → вставиться "Исправь [вміст буфера]"

## Аліаси для терміналу

Додайте до `~/.bashrc` або `~/.zshrc`:

```bash
alias rec='/шлях/до/recognize'
```

## Як це працює

1. Слухає події клавіші F9 через системні хуки
2. Записує аудіо з мікрофона за замовчуванням у WAV-файл
3. Надсилає аудіо до OpenAI Whisper API для розпізнавання
4. Обробляє текст (прибирає три крапки в кінці, підставляє вказівні фрази)
5. Вставляє результат через Ctrl+V, зберігаючи вихідний буфер обміну

## Ліцензія

MIT
