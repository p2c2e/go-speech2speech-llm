package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type TextToSpeech struct {
	apiKey    string
	modelName string
}

type MessageType string

const (
	TypeSpeak    MessageType = "Speak"
	TypeFlush    MessageType = "Flush"
	TypeClear    MessageType = "Clear"
	TypeClose    MessageType = "Close"
	TypeMetadata MessageType = "Metadata"
	TypeWarning  MessageType = "Warning"
)

type SpeakMessage struct {
	Type MessageType `json:"type"`
	Text string     `json:"text"`
}

type ControlMessage struct {
	Type MessageType `json:"type"`
}

type MetadataMessage struct {
	Type         MessageType `json:"type"`
	RequestID    string      `json:"request_id"`
	ModelName    string      `json:"model_name"`
	ModelVersion string      `json:"model_version"`
	ModelUUID    string      `json:"model_uuid"`
}

type WarningMessage struct {
	Type     MessageType `json:"type"`
	WarnCode string      `json:"warn_code"`
	WarnMsg  string      `json:"warn_msg"`
}

func NewTextToSpeech() *TextToSpeech {
	return &TextToSpeech{
		apiKey:    os.Getenv("DEEPGRAM_API_KEY"),
		modelName: "aura-asteria-en",
	}
}

func (t *TextToSpeech) IsInstalled(libName string) bool {
	_, err := exec.LookPath(libName)
	return err == nil
}

func (t *TextToSpeech) Speak(text string) error {
	if !t.IsInstalled("ffplay") {
		return fmt.Errorf("ffplay not found, necessary to stream audio")
	}

	// Prepare the HTTP request
	apiURL := fmt.Sprintf("https://api.deepgram.com/v1/speak?model=%s&encoding=linear16&sample_rate=24000", t.modelName)
	
	payload := map[string]string{
		"text": text,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Token "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Create HTTP client
	client := &http.Client{}
	
	// Start the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned error status %d: %s", resp.StatusCode, string(body))
	}

	// Start ffplay
	ffplayArgs := []string{"-f", "wav", "-autoexit", "-", "-nodisp"}
	fmt.Printf("Running command: ffplay %v\n", ffplayArgs)
	cmd := exec.Command("ffplay", ffplayArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	devNull, err := os.OpenFile("/dev/null", os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to open /dev/null: %v", err)
	}
	defer devNull.Close()
	
	cmd.Stdout = devNull
	cmd.Stderr = devNull

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffplay: %v", err)
	}

	// Create a buffer for reading chunks
	buf := make([]byte, 1024)
	startTime := time.Now()
	firstByte := true

	// Stream the audio data to ffplay
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if firstByte {
				ttfb := time.Since(startTime).Milliseconds()
				fmt.Printf("TTS Time to First Byte (TTFB): %dms\n", ttfb)
				firstByte = false
			}
			
			if _, err := stdin.Write(buf[:n]); err != nil {
				return fmt.Errorf("failed to write audio data: %v", err)
			}
		}
		
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading response: %v", err)
		}
	}

	stdin.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffplay error: %v", err)
	}
	return nil
}
