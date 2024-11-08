package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type LanguageModelProcessor struct {
	apiKey     string
	messages   []Message
	systemPrompt string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type GroqResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func NewLanguageModelProcessor() (*LanguageModelProcessor, error) {
	systemPrompt, err := os.ReadFile("system_prompt.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to read system prompt: %v", err)
	}

	return &LanguageModelProcessor{
		apiKey:       os.Getenv("GROQ_API_KEY"),
		messages:     []Message{{Role: "system", Content: string(systemPrompt)}},
		systemPrompt: string(systemPrompt),
	}, nil
}

func (l *LanguageModelProcessor) Process(text string) (string, error) {
	startTime := time.Now()

	l.messages = append(l.messages, Message{Role: "user", Content: text})

	reqBody := GroqRequest{
		Model:    "mixtral-8x7b-32768",
		Messages: l.messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	fmt.Printf("Sending request to Groq API with %d messages...\n", len(l.messages))
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read the error response body
		var errorBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorBody); err != nil {
			return "", fmt.Errorf("API error (status %d): failed to decode error response", resp.StatusCode)
		}
		return "", fmt.Errorf("API error (status %d): %v", resp.StatusCode, errorBody)
	}

	var groqResp GroqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices received from API")
	}

	response := groqResp.Choices[0].Message.Content
	l.messages = append(l.messages, Message{Role: "assistant", Content: response})

	elapsed := time.Since(startTime)
	fmt.Printf("LLM (%dms): %s\n", elapsed.Milliseconds(), response)

	return response, nil
}
