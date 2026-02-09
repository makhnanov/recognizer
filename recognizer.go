package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/gordonklaus/portaudio"
	"github.com/joho/godotenv"
	hook "github.com/robotn/gohook"
)

const (
	SampleRate      = 44100
	BitsPerSample   = 16
	NumChannels     = 1
	InputBufferSize = 64
	BeepBufferSize  = 512
	BeepDuration    = 0.25
	BeepFrequency   = 1046.50 // C6 note
	F9KeyCode       = 65478
	APITimeout      = 30 * time.Second
)

var (
	OPENAI_API_KEY   string
	WHISPER_MODEL    string
	WHISPER_LANGUAGE string
)

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
	doneChan    chan struct{}
	dataSize    int
}

// WAV header writer
func writeWavHeader(f *os.File, dataSize int) error {
	byteRate := SampleRate * NumChannels * BitsPerSample / 8
	blockAlign := NumChannels * BitsPerSample / 8
	chunkSize := 36 + dataSize

	f.Seek(0, 0)
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, uint32(chunkSize))
	f.Write([]byte("WAVE"))

	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))
	binary.Write(f, binary.LittleEndian, uint16(1))
	binary.Write(f, binary.LittleEndian, uint16(NumChannels))
	binary.Write(f, binary.LittleEndian, uint32(SampleRate))
	binary.Write(f, binary.LittleEndian, uint32(byteRate))
	binary.Write(f, binary.LittleEndian, uint16(blockAlign))
	binary.Write(f, binary.LittleEndian, uint16(BitsPerSample))

	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, uint32(dataSize))

	return nil
}

func NewAudioRecorder() *AudioRecorder {
	return &AudioRecorder{
		inputBuffer: make([]float32, InputBufferSize),
	}
}

// playBeep plays a short xylophone-like notification sound
func playBeep() {
	numSamples := int(float64(SampleRate) * BeepDuration)
	samples := make([]float32, BeepBufferSize)

	stream, err := portaudio.OpenDefaultStream(0, 1, float64(SampleRate), BeepBufferSize, samples)
	if err != nil {
		log.Printf("beep: failed to open stream: %v", err)
		return
	}
	defer stream.Close()

	if err = stream.Start(); err != nil {
		log.Printf("beep: failed to start stream: %v", err)
		return
	}
	defer stream.Stop()
	totalSamples := 0
	for totalSamples < numSamples {
		samplesInBuffer := BeepBufferSize
		if totalSamples+samplesInBuffer > numSamples {
			samplesInBuffer = numSamples - totalSamples
		}

		for i := 0; i < samplesInBuffer; i++ {
			t := float64(totalSamples+i) / float64(SampleRate)
			envelope := math.Exp(-4.5 * t / BeepDuration)
			fundamental := math.Sin(2 * math.Pi * BeepFrequency * t)
			harmonic2 := 0.3 * math.Sin(2*math.Pi*BeepFrequency*2*t)
			harmonic3 := 0.15 * math.Sin(2*math.Pi*BeepFrequency*3*t)
			harmonic4 := 0.08 * math.Sin(2*math.Pi*BeepFrequency*4*t)
			sample := fundamental + harmonic2 + harmonic3 + harmonic4
			samples[i] = float32(0.25 * sample * envelope)
		}

		if err = stream.Write(); err != nil {
			log.Printf("beep: write error: %v", err)
			return
		}

		totalSamples += samplesInBuffer
	}

	time.Sleep(time.Duration(BeepDuration*1000) * time.Millisecond)
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

	stream, err := portaudio.OpenDefaultStream(1, 0, SampleRate, len(ar.inputBuffer), ar.inputBuffer)
	if err != nil {
		return err
	}
	ar.stream = stream

	if err := ar.stream.Start(); err != nil {
		return err
	}

	ar.recording = true
	ar.stopChan = make(chan struct{})
	ar.doneChan = make(chan struct{})

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
	defer close(ar.doneChan)

	for {
		select {
		case <-ar.stopChan:
			return
		default:
			if err := ar.stream.Read(); err != nil {
				log.Printf("stream read error: %v", err)
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

// transcribeAudio sends audio file to OpenAI Whisper API for speech recognition
func transcribeAudio(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %v", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %v", err)
	}

	if _, err = io.Copy(fileWriter, file); err != nil {
		return "", fmt.Errorf("failed to copy file data: %v", err)
	}

	writer.WriteField("model", WHISPER_MODEL)
	if WHISPER_LANGUAGE != "" {
		writer.WriteField("language", WHISPER_LANGUAGE)
	}
	writer.Close()

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/transcriptions", &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+OPENAI_API_KEY)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: APITimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var whisperResp WhisperResponse
	if err = json.Unmarshal(body, &whisperResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	return strings.TrimSpace(whisperResp.Text), nil
}

// copyToClipboardAndPaste copies text to clipboard, pastes it, and restores previous clipboard content
func copyToClipboardAndPaste(text string) error {
	if text == "" {
		return nil
	}

	previousClipboard, _ := robotgo.ReadAll()

	robotgo.WriteAll(text)
	time.Sleep(100 * time.Millisecond)

	robotgo.KeyTap("v", "ctrl")

	time.Sleep(200 * time.Millisecond)

	robotgo.WriteAll(previousClipboard)

	return nil
}

func main() {
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal("failed to get executable path: ", err)
	}
	execDir := filepath.Dir(execPath)

	if err := godotenv.Load(filepath.Join(execDir, ".env")); err != nil {
		log.Fatalf("failed to load .env: %v", err)
	}

	OPENAI_API_KEY = os.Getenv("OPENAI_API_KEY")
	if OPENAI_API_KEY == "" {
		log.Fatal("OPENAI_API_KEY not set")
	}

	WHISPER_MODEL = os.Getenv("WHISPER_MODEL")
	if WHISPER_MODEL == "" {
		WHISPER_MODEL = "gpt-4o-transcribe"
	}

	WHISPER_LANGUAGE = os.Getenv("WHISPER_LANGUAGE")

	portaudio.Initialize()
	defer portaudio.Terminate()

	voicesDir := filepath.Join(execDir, ".voices")
	if err := os.MkdirAll(voicesDir, 0755); err != nil {
		log.Fatalf("failed to create .voices dir: %v", err)
	}

	recorder := NewAudioRecorder()
	var currentFilename string

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	evChan := hook.Start()
	defer hook.End()

	phrasesFile := filepath.Join(execDir, "phrases.txt")

	fmt.Println("Hold F9 to record, release to stop and transcribe.")

loop:
	for {
		select {
		case ev := <-evChan:
			if ev.Kind == hook.KeyDown && ev.Rawcode == F9KeyCode && !recorder.recording {
				playBeep()
				currentFilename = filepath.Join(voicesDir, fmt.Sprintf("voice_%s.wav", time.Now().Format("2006-01-02_15-04-05")))
				log.Printf("recording: %s", currentFilename)
				if err := recorder.StartRecording(currentFilename); err != nil {
					log.Printf("start error: %v", err)
				}
			}
			if ev.Kind == hook.KeyUp && ev.Rawcode == F9KeyCode && recorder.recording {
				log.Print("stopped")
				if err := recorder.StopRecording(); err != nil {
					log.Printf("stop error: %v", err)
					continue
				}
				playBeep()

				text, err := transcribeAudio(currentFilename)
				if err != nil {
					log.Printf("transcribe error: %v", err)
					continue
				}
				log.Printf("text: %s", text)

				// Remove trailing ellipsis
				textToInsert := text
				if strings.HasSuffix(text, "...") {
					textToInsert = strings.TrimSuffix(text, "...") + " "
				}

				// Replace pointer phrases with clipboard content
				phrases, err := loadPhrases(phrasesFile)
				if err != nil {
					log.Printf("load phrases error: %v", err)
				} else {
					textToInsert = replacePointerPhrases(textToInsert, phrases)
				}

				copyToClipboardAndPaste(textToInsert)
			}
		case <-sigs:
			break loop
		}
	}
}

// loadPhrases reads pointer phrases from a text file (one phrase per line)
func loadPhrases(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var phrases []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			phrases = append(phrases, line)
		}
	}
	return phrases, nil
}

// replacePointerPhrases replaces Russian pointer phrases with clipboard content
func replacePointerPhrases(text string, phrases []string) string {
	lowerText := strings.ToLower(text)
	hasPhrase := false
	for _, p := range phrases {
		if strings.Contains(lowerText, p) {
			hasPhrase = true
			break
		}
	}
	if !hasPhrase {
		return text
	}

	clipboardContent, err := robotgo.ReadAll()
	if err != nil {
		return text
	}

	for _, phrase := range phrases {
		text = strings.ReplaceAll(text, phrase, clipboardContent)
		capitalized := strings.ToUpper(string([]rune(phrase)[0])) + string([]rune(phrase)[1:])
		text = strings.ReplaceAll(text, capitalized, clipboardContent)
		text = strings.ReplaceAll(text, strings.ToUpper(phrase), clipboardContent)
	}
	return text
}
