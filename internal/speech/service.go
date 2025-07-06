package speech

import (
	"context"
	"fmt"
	"io"
	"log"

	speech "cloud.google.com/go/speech/apiv2"
	speechpb "cloud.google.com/go/speech/apiv2/speechpb"
)

// Service handles speech-to-text operations using Google Cloud Speech-to-Text v2 API
type Service struct {
	client    *speech.Client
	projectID string
	debug     bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewService creates a new speech service
func NewService(projectID string, debug bool) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	client, err := speech.NewClient(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create speech client: %w", err)
	}

	return &Service{
		client:    client,
		projectID: projectID,
		debug:     debug,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// StreamingRecognizeConfig creates the configuration for streaming recognition
func (s *Service) createStreamingConfig() *speechpb.StreamingRecognitionConfig {
	return &speechpb.StreamingRecognitionConfig{
		Config: &speechpb.RecognitionConfig{
			// Use auto-detect for audio encoding
			DecodingConfig: &speechpb.RecognitionConfig_AutoDecodingConfig{
				AutoDecodingConfig: &speechpb.AutoDetectDecodingConfig{},
			},
			LanguageCodes: []string{"en-US"},
			Model:         "latest_long", // Chirp model for conversational speech
			Features: &speechpb.RecognitionFeatures{
				EnableAutomaticPunctuation: true,
				EnableWordConfidence:       true,
				EnableWordTimeOffsets:      true,
			},
		},
		StreamingFeatures: &speechpb.StreamingRecognitionFeatures{
			EnableVoiceActivityEvents: true,
			InterimResults:            true,
		},
	}
}

// StartStreaming creates a new streaming recognition session
func (s *Service) StartStreaming() (*StreamingSession, error) {
	recognizer := fmt.Sprintf("projects/%s/locations/global/recognizers/_", s.projectID)

	stream, err := s.client.StreamingRecognize(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create streaming recognize: %w", err)
	}

	// Send the initial configuration
	config := s.createStreamingConfig()
	configRequest := &speechpb.StreamingRecognizeRequest{
		Recognizer: recognizer,
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: config,
		},
	}

	if err := stream.Send(configRequest); err != nil {
		return nil, fmt.Errorf("failed to send config: %w", err)
	}

	session := &StreamingSession{
		stream:  stream,
		service: s,
	}

	// Start listening for responses
	go session.listen()

	return session, nil
}

// Close closes the speech service
func (s *Service) Close() error {
	s.cancel()
	return s.client.Close()
}

// StreamingSession represents an active streaming recognition session
type StreamingSession struct {
	stream     speechpb.Speech_StreamingRecognizeClient
	service    *Service
	ResultChan chan *TranscriptionResult
}

// TranscriptionResult contains the transcription results
type TranscriptionResult struct {
	Transcript  string
	Confidence  float32
	IsFinal     bool
	Speaker     int32
	WordDetails []*speechpb.WordInfo
	Language    string
}

// SendAudio sends PCM audio data to the streaming session
func (s *StreamingSession) SendAudio(audioData []byte) error {
	request := &speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_Audio{
			Audio: audioData,
		},
	}

	return s.stream.Send(request)
}

// listen listens for streaming responses
func (s *StreamingSession) listen() {
	s.ResultChan = make(chan *TranscriptionResult, 10)
	defer close(s.ResultChan)

	for {
		response, err := s.stream.Recv()
		if err == io.EOF {
			if s.service.debug {
				log.Printf("Speech streaming ended")
			}
			return
		}
		if err != nil {
			if s.service.debug {
				log.Printf("Error receiving from speech stream: %v", err)
			}
			return
		}

		// Process the response
		if response.Results != nil {
			for _, result := range response.Results {
				if len(result.Alternatives) > 0 {
					alt := result.Alternatives[0]

					transcriptionResult := &TranscriptionResult{
						Transcript:  alt.Transcript,
						Confidence:  alt.Confidence,
						IsFinal:     result.IsFinal,
						WordDetails: alt.Words,
						Language:    result.LanguageCode,
					}

					// Extract speaker information if available
					if len(alt.Words) > 0 && alt.Words[0].SpeakerLabel != "" {
						// Parse speaker label (usually in format "speaker_0", "speaker_1", etc.)
						var speaker int32
						if _, err := fmt.Sscanf(alt.Words[0].SpeakerLabel, "speaker_%d", &speaker); err == nil {
							transcriptionResult.Speaker = speaker
						}
					}

					select {
					case s.ResultChan <- transcriptionResult:
						if s.service.debug {
							finalText := ""
							if transcriptionResult.IsFinal {
								finalText = " [FINAL]"
							}
							log.Printf("Speech result%s: %s (confidence: %.2f)",
								finalText, transcriptionResult.Transcript, transcriptionResult.Confidence)
						}
					default:
						// Channel is full, skip this result
						if s.service.debug {
							log.Printf("Speech result channel full, dropping result")
						}
					}
				}
			}
		}
	}
}

// Close closes the streaming session
func (s *StreamingSession) Close() error {
	if s.ResultChan != nil {
		close(s.ResultChan)
	}
	return s.stream.CloseSend()
}
