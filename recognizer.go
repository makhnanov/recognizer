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
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
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
	OPENAI_API_KEY       string
	WHISPER_MODEL        string
	WHISPER_LANGUAGE     string
	FILTER_MODEL         string
	NEGATIVITY_THRESHOLD int
)

type WhisperResponse struct {
	Text string `json:"text"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	Temperature    float64         `json:"temperature"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type ChatChoice struct {
	Message ChatMessage `json:"message"`
}

type ChatResponse struct {
	Choices []ChatChoice `json:"choices"`
}

type NegativityResult struct {
	NegativityPercent int    `json:"negativity_percent"`
	RewrittenText     string `json:"rewritten_text"`
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

// filterNegativity analyzes text negativity via OpenAI Chat API.
// If negativity exceeds NEGATIVITY_THRESHOLD, returns a polite rewritten version.
func filterNegativity(text string) (string, error) {
	systemPrompt := `You are a text analysis assistant. Analyze the negativity level of the given text. ` +
		`Rate it from 0 to 100 where 0 is completely positive/neutral and 100 is extremely negative/rude/offensive. ` +
		`Always rewrite the text in a polite, politically correct way while preserving the original meaning and language. ` +
		`Respond with JSON: {"negativity_percent": <0-100>, "rewritten_text": "<polite version of the text in the same language>"}`

	reqBody := ChatRequest{
		Model: FILTER_MODEL,
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: text},
		},
		Temperature:    0.3,
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return text, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return text, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+OPENAI_API_KEY)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: APITimeout}
	resp, err := client.Do(req)
	if err != nil {
		return text, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return text, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return text, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err = json.Unmarshal(body, &chatResp); err != nil {
		return text, fmt.Errorf("failed to parse response: %v", err)
	}

	if len(chatResp.Choices) == 0 {
		return text, fmt.Errorf("no choices in response")
	}

	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)

	var result NegativityResult
	if err = json.Unmarshal([]byte(content), &result); err != nil {
		return text, fmt.Errorf("failed to parse negativity result: %v (content: %s)", err, content)
	}

	log.Printf("negativity: %d%% (threshold: %d%%)", result.NegativityPercent, NEGATIVITY_THRESHOLD)

	if result.NegativityPercent > NEGATIVITY_THRESHOLD {
		log.Printf("text rewritten from [%s] to [%s]", text, result.RewrittenText)
		return result.RewrittenText, nil
	}

	return text, nil
}

// getActiveWindowClass returns WM_CLASS of the currently focused window
func getActiveWindowClass() string {
	out, err := exec.Command("xprop", "-root", "_NET_ACTIVE_WINDOW").Output()
	if err != nil {
		log.Printf("xprop root failed: %v", err)
		return ""
	}
	log.Printf("xprop root: %s", strings.TrimSpace(string(out)))

	// Parse: _NET_ACTIVE_WINDOW(WINDOW): window id # 0xa800003, 0x0
	raw := string(out)
	idx := strings.Index(raw, "# ")
	if idx == -1 {
		log.Print("xprop: no window id found")
		return ""
	}
	rest := strings.TrimSpace(raw[idx+2:])
	windowID := strings.TrimRight(strings.Fields(rest)[0], ",")
	log.Printf("active window ID: %s", windowID)

	out, err = exec.Command("xprop", "-id", windowID, "WM_CLASS").Output()
	if err != nil {
		log.Printf("xprop WM_CLASS failed: %v", err)
		return ""
	}

	wmClass := strings.TrimSpace(string(out))
	log.Printf("WM_CLASS: %s", wmClass)
	return wmClass
}

// isTerminalWindow checks if WM_CLASS belongs to a terminal emulator
func isTerminalWindow(wmClass string) bool {
	lower := strings.ToLower(wmClass)
	terminals := []string{
		"terminal", "konsole", "alacritty", "kitty", "tilix",
		"terminator", "guake", "yakuake", "urxvt", "rxvt",
		"xterm", "wezterm", "sakura", "terminology", "st-256color",
	}
	for _, t := range terminals {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

// copyToClipboardAndPaste copies text to clipboard, pastes it, and restores previous clipboard content
func copyToClipboardAndPaste(text string) error {
	if text == "" {
		return nil
	}

	previousClipboard, _ := robotgo.ReadAll()

	robotgo.WriteAll(text)
	time.Sleep(100 * time.Millisecond)

	wmClass := getActiveWindowClass()
	if isTerminalWindow(wmClass) {
		log.Print("terminal detected, using ctrl+shift+v")
		robotgo.KeyTap("v", "ctrl", "shift")
	} else {
		log.Print("non-terminal window, using ctrl+v")
		robotgo.KeyTap("v", "ctrl")
	}

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

	FILTER_MODEL = os.Getenv("FILTER_MODEL")
	if FILTER_MODEL == "" {
		FILTER_MODEL = "gpt-4o-mini"
	}

	NEGATIVITY_THRESHOLD = 50
	if v := os.Getenv("NEGATIVITY_THRESHOLD"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			NEGATIVITY_THRESHOLD = parsed
		}
	}

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

				// Filter negativity — rewrite if too negative
				if NEGATIVITY_THRESHOLD > 0 {
					filteredText, filterErr := filterNegativity(textToInsert)
					if filterErr != nil {
						log.Printf("negativity filter error: %v", filterErr)
					} else {
						textToInsert = filteredText
					}
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

// replacePointerPhrases replaces Russian pointer phrases with clipboard content.
// Phrases are sorted by length descending so longer matches are replaced first.
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

	sorted := make([]string, len(phrases))
	copy(sorted, phrases)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i]) > len(sorted[j])
	})

	for _, phrase := range sorted {
		text = strings.ReplaceAll(text, phrase, clipboardContent)
		capitalized := strings.ToUpper(string([]rune(phrase)[0])) + string([]rune(phrase)[1:])
		text = strings.ReplaceAll(text, capitalized, clipboardContent)
		text = strings.ReplaceAll(text, strings.ToUpper(phrase), clipboardContent)
	}
	return text
}
