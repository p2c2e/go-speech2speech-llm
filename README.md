# Go Speech-to-Speech Conversation System

A real-time speech-to-speech conversation system built in Go that uses:
- Deepgram for speech-to-text transcription and text-to-speech synthesis
- Groq for LLM-powered conversation responses
- FFplay for audio playback

## Prerequisites

- Go 1.x
- FFplay (part of FFmpeg)
- Deepgram API key
- Groq API key

## Setup

1. Clone the repository
2. Copy `env.sample` to `.env` and fill in your API keys:
   ```
   DEEPGRAM_API_KEY=your_key_here
   GROQ_API_KEY=your_key_here
   ```
3. Install dependencies:
   ```
   go mod download
   ```

## Usage

Run the application:
```
go run main.go
```

The system will:
1. Listen for your speech input
2. Transcribe it to text
3. Process it through the LLM
4. Convert the response to speech
5. Play the audio response

## Architecture

- `conversation.go`: Main conversation loop and component coordination
- `transcription.go`: Speech-to-text using Deepgram's WebSocket API
- `llm.go`: Text processing using Groq's LLM API
- `tts.go`: Text-to-speech synthesis using Deepgram's API

## License

[Add your chosen license]
