package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/gordonklaus/portaudio"
	"github.com/joho/godotenv"
	hook "github.com/robotn/gohook"
)

var OPENAI_API_KEY string

type WhisperResponse struct {
	Text string `json:"text"`
}

type AudioRecorder struct {
	stream      *portaudio.Stream
	outputFile  *os.File
	recording   bool
	mutex       sync.Mutex
	inputBuffer []float32
	stopChan    chan struct{}
	doneChan    chan struct{} // FIX: ждём завершения горутины
	dataSize    int
}

// WAV header writer
func writeWavHeader(f *os.File, dataSize int) error {
	sampleRate := 44100
	bitsPerSample := 16
	numChannels := 1

	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	chunkSize := 36 + dataSize

	f.Seek(0, 0)
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(chunkSize))
	f.Write([]byte("WAVE"))

	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1))
	binary.Write(f, binary.LittleEndian, uint16(numChannels))
	binary.Write(f, binary.LittleEndian, uint32(sampleRate))
	binary.Write(f, binary.LittleEndian, uint32(byteRate))
	binary.Write(f, binary.LittleEndian, uint16(blockAlign))
	binary.Write(f, binary.LittleEndian, uint16(bitsPerSample))

	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataSize))

	return nil
}

func NewAudioRecorder() *AudioRecorder {
	return &AudioRecorder{
		inputBuffer: make([]float32, 64),
	}
}

func (ar *AudioRecorder) StartRecording(filename string) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if ar.recording {
		return fmt.Errorf("already recording")
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	ar.outputFile = file
	ar.dataSize = 0
	writeWavHeader(file, 0)

	stream, err := portaudio.OpenDefaultStream(1, 0, 44100, len(ar.inputBuffer), ar.inputBuffer)
	if err != nil {
		return err
	}
	ar.stream = stream

	if err := ar.stream.Start(); err != nil {
		return err
	}

	ar.recording = true
	ar.stopChan = make(chan struct{})
	ar.doneChan = make(chan struct{}) // FIX

	go ar.recordingLoop()

	return nil
}

func (ar *AudioRecorder) StopRecording() error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if !ar.recording {
		return nil
	}

	close(ar.stopChan)

	// FIX: ждём завершения горутины
	<-ar.doneChan

	if ar.stream != nil {
		ar.stream.Stop()
		ar.stream.Close()
		ar.stream = nil
	}

	if ar.outputFile != nil {
		writeWavHeader(ar.outputFile, ar.dataSize)
		ar.outputFile.Close()
		ar.outputFile = nil
	}

	ar.recording = false
	return nil
}

func (ar *AudioRecorder) recordingLoop() {
	defer close(ar.doneChan) // FIX: сигнализируем, что горутина завершена

	for {
		select {
		case <-ar.stopChan:
			return
		default:
			err := ar.stream.Read()
			if err != nil {
				fmt.Printf("Error reading from stream: %v\n", err)
				return
			}
			for _, sample := range ar.inputBuffer {
				intSample := int16(sample * 32767)
				binary.Write(ar.outputFile, binary.LittleEndian, intSample)
				ar.dataSize += 2
			}
		}
	}
}

// transcribeAudio отправляет аудио файл в OpenAI Whisper API для распознавания речи
func transcribeAudio(filename string) (string, error) {
	if OPENAI_API_KEY == "YOUR_OPENAI_API_KEY_HERE" {
		return "", fmt.Errorf("OpenAI API ключ не установлен. Измените OPENAI_API_KEY в коде")
	}

	// Открываем аудио файл
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %v", err)
	}
	defer file.Close()

	// Создаем multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Добавляем файл
	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %v", err)
	}

	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file data: %v", err)
	}

	// Добавляем модель
	writer.WriteField("model", "whisper-1")

	// Добавляем язык для лучшего распознавания русского
	writer.WriteField("language", "ru")

	writer.Close()

	// Создаем HTTP запрос
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+OPENAI_API_KEY)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Отправляем запрос
	fmt.Printf("[%s] 🔄 Отправка запроса в Whisper API...\n", time.Now().Format("15:04:05"))
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[%s] ✓ Получен ответ от Whisper API (HTTP %d)\n", time.Now().Format("15:04:05"), resp.StatusCode)

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Парсим JSON ответ
	var whisperResp WhisperResponse
	err = json.Unmarshal(body, &whisperResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	return strings.TrimSpace(whisperResp.Text), nil
}

// copyToClipboardAndPaste копирует текст в буфер обмена и вставляет его в позицию курсора
func copyToClipboardAndPaste(text string) error {
	if text == "" {
		return nil
	}

	// Пробуем установить содержимое буфера обмена
	robotgo.WriteAll(text)

	// Небольшая задержка
	time.Sleep(100 * time.Millisecond)

	// Проверим, что в буфере обмена
	clipContent, err := robotgo.ReadAll()
	if err != nil || clipContent != text {
		// Если не получилось записать в буфер обмена, печатаем текст напрямую
		fmt.Println("⚠️ Буфер обмена недоступен, вводим текст напрямую...")
		robotgo.TypeStr(text)
		return nil
	}

	// Если в буфере обмена правильный текст, вставляем его
	robotgo.KeyTap("v", "ctrl")

	return nil
}

func main() {
	// Загружаем переменные окружения из .env файла
	if err := godotenv.Load(); err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	// Получаем API ключ из переменной окружения
	OPENAI_API_KEY = os.Getenv("OPENAI_API_KEY")
	if OPENAI_API_KEY == "" {
		log.Fatal("OPENAI_API_KEY не установлен в .env файле")
	}

	portaudio.Initialize()
	defer portaudio.Terminate()

	// Создаём папку .voices если её нет
	if err := os.MkdirAll(".voices", 0755); err != nil {
		fmt.Printf("Ошибка создания папки .voices: %v\n", err)
		return
	}

	recorder := NewAudioRecorder()
	var currentFilename string
	var middleButtonPressed bool
	var middleButtonRecordingStarted bool

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	evChan := hook.Start()
	defer hook.End()

	fmt.Println("Управление: зажать F9 или среднюю кнопку мыши (>0.5с) = запись, отпустить = стоп.")
	fmt.Println("После записи аудио будет автоматически распознано с помощью Whisper API.")

	// Горутина для проверки длительности удержания средней кнопки
	checkMiddleButtonTicker := time.NewTicker(100 * time.Millisecond)
	defer checkMiddleButtonTicker.Stop()

loop:
	for {
		select {
		case <-checkMiddleButtonTicker.C:
			// Проверяем, зажата ли средняя кнопка более 0.5 секунды
			//if middleButtonPressed && !middleButtonRecordingStarted {
			//	pressDuration := time.Since(middleButtonPressTime)
			//	if pressDuration >= 500*time.Millisecond {
			//		middleButtonRecordingStarted = true
			//		timestamp := time.Now().Format("2006-01-02_15-04-05")
			//		currentFilename = fmt.Sprintf(".voices/voice_recording_%s.wav", timestamp)
			//		fmt.Printf("[%s] ▶️ Старт записи: %s\n", time.Now().Format("15:04:05"), currentFilename)
			//		if err := recorder.StartRecording(currentFilename); err != nil {
			//			fmt.Println("Ошибка:", err)
			//		}
			//	}
			//}

		case ev := <-evChan:

			// Отслеживаем нажатие F9 (Rawcode 65478)
			if ev.Kind == hook.KeyDown && ev.Rawcode == 65478 { // F9
				if !recorder.recording {
					timestamp := time.Now().Format("2006-01-02_15-04-05")
					currentFilename = fmt.Sprintf(".voices/voice_recording_%s.wav", timestamp)
					fmt.Printf("[%s] ▶️ Старт записи: %s\n", time.Now().Format("15:04:05"), currentFilename)
					if err := recorder.StartRecording(currentFilename); err != nil {
						fmt.Println("Ошибка:", err)
					}
				}
			}
			if ev.Kind == hook.KeyUp && ev.Rawcode == 65478 { // F9
				if recorder.recording {
					fmt.Printf("[%s] ⏹ Стоп записи\n", time.Now().Format("15:04:05"))
					if err := recorder.StopRecording(); err != nil {
						fmt.Println("Ошибка:", err)
					} else if currentFilename != "" {
						// Распознаем речь
						text, err := transcribeAudio(currentFilename)
						if err != nil {
							fmt.Printf("❌ Ошибка распознавания: %v\n", err)
						} else {
							fmt.Printf("[%s] 📝 %s\n", time.Now().Format("15:04:05"), text)

							// Переименовываем файл, добавляя распознанный текст
							timestamp := time.Now().Unix()
							newFilename := fmt.Sprintf(".voices/%s_%s.wav", timestamp, text)
							if renameErr := os.Rename(currentFilename, newFilename); renameErr != nil {
								fmt.Printf("⚠️ Не удалось переименовать файл: %v\n", renameErr)
							}

							// Копируем в буфер обмена и вставляем в позицию курсора
							if copyErr := copyToClipboardAndPaste(text); copyErr != nil {
								fmt.Printf("❌ Ошибка копирования/вставки: %v\n", copyErr)
							}
						}
					}
				}
			}

			// Отслеживаем нажатие средней кнопки мыши (Button 2)
			if ev.Kind == hook.MouseDown && ev.Button == 2 {
				middleButtonPressed = true
				middleButtonRecordingStarted = false
				//middleButtonPressTime = time.Now()
			}

			// Отслеживаем отпускание средней кнопки мыши (Button 2)
			if ev.Kind == hook.MouseUp && ev.Button == 2 {
				// Если запись была начата средней кнопкой, останавливаем её
				if middleButtonPressed && middleButtonRecordingStarted && recorder.recording {
					fmt.Printf("[%s] ⏹ Стоп записи\n", time.Now().Format("15:04:05"))
					if err := recorder.StopRecording(); err != nil {
						fmt.Println("Ошибка:", err)
					} else if currentFilename != "" {
						// Распознаем речь
						text, err := transcribeAudio(currentFilename)
						if err != nil {
							fmt.Printf("❌ Ошибка распознавания: %v\n", err)
						} else {
							fmt.Printf("[%s] 📝 %s\n", time.Now().Format("15:04:05"), text)

							// Переименовываем файл, добавляя распознанный текст
							timestamp := time.Now().Format("2006-01-02_15-04-05")
							newFilename := fmt.Sprintf(".voices/%s_%s.wav", timestamp, text)
							if renameErr := os.Rename(currentFilename, newFilename); renameErr != nil {
								fmt.Printf("⚠️ Не удалось переименовать файл: %v\n", renameErr)
							}

							// Копируем в буфер обмена и вставляем в позицию курсора
							if copyErr := copyToClipboardAndPaste(text); copyErr != nil {
								fmt.Printf("❌ Ошибка копирования/вставки: %v\n", copyErr)
							}
						}
					}
				}

				middleButtonPressed = false
				middleButtonRecordingStarted = false
			}

			//if ev.Kind == hook.KeyDown && ev.Keycode == 1 { // Esc
			//	fmt.Println("Выход")
			//	break loop
			//}
		case <-sigs:
			break loop
		}
	}
}
