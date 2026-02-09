![vibe](https://img.shields.io/badge/vibe-clauded-da6a46)

# Recognizer

**أداة إدخال الصوت إلى نص لنظام Linux** — اضغط مع الاستمرار على مفتاح، تحدث، أفلت، وستظهر كلماتك عند موضع المؤشر.

[Русский](Readme.ru.md) | [English](Readme.md) | [Қазақша](Readme.kk.md) | [Українська](Readme.uk.md) | [日本語](Readme.ja.md) | [中文](Readme.zh.md)

## الميزات

- **التسجيل بالضغط للتحدث** — اضغط مع الاستمرار على F9 للتسجيل، أفلت للنسخ
- **إدراج النص الفوري** — يتم لصق النص المنسوخ تلقائياً في موضع المؤشر
- **الحفاظ على الحافظة** — يتم حفظ محتوى الحافظة واستعادته بعد اللصق
- **التغذية الراجعة الصوتية** — صوت إكسيلوفون عند بدء/إيقاف التسجيل
- **استبدال العبارات الإشارية** — قل "вот это" لإدراج محتوى الحافظة

## المتطلبات

- نظام Linux مع X11
- Go 1.21+
- مكتبة PortAudio
- مفتاح OpenAI API

## التثبيت

### 1. تثبيت التبعيات

```bash
# Ubuntu/Debian
sudo apt install portaudio19-dev libx11-dev libxtst-dev libxinerama-dev libxcursor-dev libxkbcommon-dev

# Fedora
sudo dnf install portaudio-devel libX11-devel libXtst-devel libXinerama-devel libXcursor-devel libxkbcommon-devel

# Arch
sudo pacman -S portaudio libx11 libxtst libxinerama libxcursor libxkbcommon
```

### 2. الاستنساخ والبناء

```bash
git clone https://github.com/makhnanov/recognizer.git
cd recognizer
cp .env.example .env
```

### 3. التكوين

حرر `.env` وأضف مفتاح OpenAI API الخاص بك:

```
OPENAI_API_KEY=sk-...
```

### 4. البناء

```bash
make
```

أو يدوياً:

```bash
go build -o recognizer recognizer.go
```

## الاستخدام

```bash
./recognizer
```

- **اضغط مع الاستمرار على F9** — بدء التسجيل (صوت تنبيه)
- **أفلت F9** — إيقاف التسجيل، النسخ، واللصق (صوت تنبيه)
- **Ctrl+C** — خروج

### العبارات الإشارية

عندما تقول عبارات مثل "вот это"، "вот эта"، "вот сюда" إلخ، سيتم استبدالها بمحتوى الحافظة الحالي. هذا يسمح لك بالإشارة إلى النص المنسوخ في الإملاء.

مثال: انسخ بعض الكود، ثم قل "Исправь вот это" → يلصق "Исправь [محتوى الحافظة]"

## الاختصارات للطرفية

أضف إلى `~/.bashrc` أو `~/.zshrc`:

```bash
alias rec='/path/to/recognizer'
```

## كيف يعمل

1. يستمع لأحداث مفتاح F9 عبر خطافات النظام
2. يسجل الصوت من الميكروفون الافتراضي إلى ملف WAV
3. يرسل الصوت إلى OpenAI Whisper API للنسخ
4. يعالج النص (يزيل النقاط الثلاث في النهاية، يستبدل العبارات الإشارية)
5. يلصق النتيجة عبر Ctrl+V، مع الحفاظ على الحافظة الأصلية

## الرخصة

MIT
