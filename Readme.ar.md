![vibe](https://img.shields.io/badge/vibe-clauded-da6a46)

# Recognizer

**أداة إدخال الصوت إلى نص لنظام Linux** — اضغط مع الاستمرار على مفتاح، تحدث، أفلت، وستظهر كلماتك عند موضع المؤشر.

[Русский](Readme.ru.md) | [English](Readme.md) | [Қазақша](Readme.kk.md) | [Українська](Readme.uk.md) | [日本語](Readme.ja.md) | [中文](Readme.zh.md)

## الميزات

- **التسجيل بالضغط للتحدث** — اضغط مع الاستمرار على F9 للتسجيل، أفلت للنسخ
- **إدراج النص الفوري** — يتم لصق النص المنسوخ تلقائياً في موضع المؤشر
- **اللصق الذكي** — يستخدم تلقائياً Ctrl+Shift+V في الطرفيات، وCtrl+V في التطبيقات الأخرى
- **الحفاظ على الحافظة** — يتم حفظ محتوى الحافظة واستعادته بعد اللصق
- **التغذية الراجعة الصوتية** — صوت إكسيلوفون عند بدء/إيقاف التسجيل
- **استبدال العبارات الإشارية** — قل "هذا" لإدراج محتوى الحافظة
- **عبارات متعددة اللغات** — عبارات إشارية بالروسية والإنجليزية والأوكرانية والكازاخية واليابانية والصينية والعربية
- **نموذج قابل للتكوين** — اختيار أي نموذج نسخ من OpenAI عبر `.env`

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

حرر `.env` واضبط المعلمات:

```
OPENAI_API_KEY=sk-...
WHISPER_MODEL=gpt-4o-transcribe
WHISPER_LANGUAGE=ar
```

- `OPENAI_API_KEY` — مفتاح OpenAI API الخاص بك (مطلوب)
- `WHISPER_MODEL` — نموذج النسخ (الافتراضي: `gpt-4o-transcribe`)
- `WHISPER_LANGUAGE` — تلميح اللغة بتنسيق ISO-639-1 (اختياري، مثل `ar`، `en`)

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

يتم تحميل العبارات الإشارية من ملف `phrases.txt` (عبارة واحدة في كل سطر، أسطر `#` هي تعليقات). يتم إعادة قراءة الملف مع كل عملية نسخ، لذا يمكنك تحريره بدون إعادة تشغيل البرنامج.

تتضمن العبارات الجاهزة عدة لغات: الروسية والإنجليزية والأوكرانية والكازاخية واليابانية والصينية والعربية.

مثال: انسخ بعض الكود، ثم قل "أصلح هذا" → يلصق "أصلح [محتوى الحافظة]"

## التشغيل التلقائي

لتشغيل Recognizer تلقائياً عند تسجيل الدخول، انسخ ملف إدخال سطح المكتب:

```bash
cp scripts/recognizer.desktop $HOME/.config/autostart/ # If executor in /var/www/recognizer/recognizer
```

## إعادة البناء وإعادة التشغيل

لإيقاف النسخة الحالية وإعادة البناء والتشغيل في الخلفية:

```bash
./scripts/restart.sh
```

يجد السكربت العملية الجارية ويوقفها، ويبني ملفاً ثنائياً جديداً، ويشغله منفصلاً عن الطرفية. يتم تسجيل المخرجات في `recognizer.log`.

## الاختصارات للطرفية

أضف إلى `~/.bashrc` أو `~/.zshrc`:

```bash
alias rec='/path/to/recognizer'
```

## كيف يعمل

1. يستمع لأحداث مفتاح F9 عبر خطافات النظام
2. يسجل الصوت من الميكروفون الافتراضي إلى ملف WAV
3. يرسل الصوت إلى API نسخ OpenAI (نموذج قابل للتكوين)
4. يعالج النص (يزيل النقاط الثلاث في النهاية، يستبدل العبارات الإشارية من `phrases.txt`)
5. يكتشف نوع النافذة النشطة ويلصق النتيجة عبر Ctrl+Shift+V (الطرفيات) أو Ctrl+V (التطبيقات الأخرى)، مع الحفاظ على الحافظة الأصلية

## الرخصة

MIT
