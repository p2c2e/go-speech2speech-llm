package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/gordonklaus/portaudio"
)

type TranscriptCollector struct {
	parts []string
}

type TranscriptionCallback func(string)

type TranscriptResponse struct {
	Channel struct {
		Alternatives []struct {
			Transcript string `json:"transcript"`
		} `json:"alternatives"`
	} `json:"channel"`
	IsFinal bool `json:"is_final"`
}

func NewTranscriptCollector() *TranscriptCollector {
	return &TranscriptCollector{
		parts: make([]string, 0),
	}
}

func (t *TranscriptCollector) Reset() {
	t.parts = t.parts[:0]
}

func (t *TranscriptCollector) AddPart(part string) {
	t.parts = append(t.parts, part)
}

func (t *TranscriptCollector) GetFullTranscript() string {
	return strings.Join(t.parts, " ")
}

func GetTranscript(ctx context.Context, callback TranscriptionCallback) error {
	u := url.URL{
		Scheme: "wss",
		Host:   "api.deepgram.com",
		Path:   "/v1/listen",
		RawQuery: url.Values{
			"model":           []string{"nova-2"},
			"punctuate":       []string{"true"},
			"language":        []string{"en-US"},
			"encoding":        []string{"linear16"},
			"channels":        []string{"1"},
			"sample_rate":     []string{"16000"},
			"interim_results": []string{"true"},
			"endpointing":     []string{"300"},
			"smart_format":    []string{"true"},
		}.Encode(),
	}

	header := make(map[string][]string)
	header["Authorization"] = []string{"Token " + os.Getenv("DEEPGRAM_API_KEY")}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %v", err)
	}
	defer c.Close()

	collector := NewTranscriptCollector()

	// Handle incoming messages
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return
				}
				fmt.Printf("read error: %v\n", err)
				return
			}

			var resp TranscriptResponse
			if err := json.Unmarshal(message, &resp); err != nil {
				fmt.Printf("json error: %v\n", err)
				continue
			}

			if len(resp.Channel.Alternatives) > 0 {
				transcript := resp.Channel.Alternatives[0].Transcript
				if transcript != "" {
					if resp.IsFinal {
						collector.AddPart(transcript)
						fullSentence := collector.GetFullTranscript()
						fmt.Printf("Human: %s\n", fullSentence)
						callback(fullSentence)
						collector.Reset()
					} else {
						fmt.Printf("Interim: %s\n", transcript)
					}
				}
			}
		}
	}()

	// Initialize PortAudio
	fmt.Println("Initializing PortAudio for microphone input...")
	if err := portaudio.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize PortAudio: %v", err)
	}
	fmt.Println("PortAudio initialized successfully")
	defer portaudio.Terminate()

	// Open default input stream
	inputChannels := 1
	sampleRate := 16000
	framesPerBuffer := make([]float32, 8192)

	fmt.Println("Opening audio input stream...")
	stream, err := portaudio.OpenDefaultStream(inputChannels, 0, float64(sampleRate), len(framesPerBuffer), framesPerBuffer)
	if err != nil {
		return fmt.Errorf("failed to open stream: %v", err)
	}
	fmt.Println("Audio input stream opened successfully")
	defer stream.Close()

	fmt.Println("Starting audio stream capture...")
	if err := stream.Start(); err != nil {
		return fmt.Errorf("failed to start stream: %v", err)
	}
	fmt.Println("Audio stream capture started - listening to microphone")

	// Convert audio samples to int16 and send over websocket
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := stream.Read(); err != nil {
					fmt.Printf("error reading from stream: %v\n", err)
					continue
				}

				// Convert float32 samples to int16
				samples := make([]int16, len(framesPerBuffer))
				for i, sample := range framesPerBuffer {
					samples[i] = int16(sample * 32767.0)
				}

				if err := c.WriteMessage(websocket.BinaryMessage, int16ToBytes(samples)); err != nil {
					fmt.Printf("error sending audio data: %v\n", err)
					return
				}
			}
		}
	}()

	<-ctx.Done()
	return nil
}

func int16ToBytes(samples []int16) []byte {
	bytes := make([]byte, len(samples)*2)
	for i, sample := range samples {
		bytes[i*2] = byte(sample)
		bytes[i*2+1] = byte(sample >> 8)
	}
	return bytes
}
