package main

import (
	"context"
	"fmt"
	"strings"
)

type ConversationManager struct {
	llm *LanguageModelProcessor
	tts *TextToSpeech
}

func NewConversationManager() *ConversationManager {
	llm, err := NewLanguageModelProcessor()
	if err != nil {
		panic(err)
	}

	return &ConversationManager{
		llm: llm,
		tts: NewTextToSpeech(),
	}
}

func (c *ConversationManager) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		var transcription string
		transcriptionDone := make(chan struct{})
		transcriptionErr := make(chan error, 1)

		go func() {
			err := GetTranscript(ctx, func(text string) {
				transcription = text
				if strings.Contains(strings.ToLower(text), "goodbye") {
					cancel()
				}
				select {
				case <-transcriptionDone:
					// Channel already closed, do nothing
				default:
					close(transcriptionDone)
				}
			})
			if err != nil {
				transcriptionErr <- err
				select {
				case <-transcriptionDone:
					// Channel already closed, do nothing
				default:
					close(transcriptionDone)
				}
			}
		}()

		select {
		case <-ctx.Done():
			return nil
		case err := <-transcriptionErr:
			fmt.Printf("Transcription error: %v\n", err)
			return err
		case <-transcriptionDone:
			if transcription != "" {
				fmt.Printf("Processing transcription with LLM: %s\n", transcription)
				response, err := c.llm.Process(transcription)
				if err != nil {
					fmt.Printf("LLM error: %v\n", err)
					continue
				}

				fmt.Printf("AI: %s\n", response)
				fmt.Println("Sending response to TTS...")
				if err := c.tts.Speak(response); err != nil {
					fmt.Printf("TTS error: %v\n", err)
				}
			}
		}
	}
}
