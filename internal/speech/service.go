package speech

import (
	"context"
	"fmt"
	"log"

	speech "cloud.google.com/go/speech/apiv1p1beta1"
	speechpb "cloud.google.com/go/speech/apiv1p1beta1/speechpb"
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

// createRecognitionConfig creates the configuration for recognition
func (s *Service) createRecognitionConfig() *speechpb.RecognitionConfig {
	return &speechpb.RecognitionConfig{
		Model:                 "latest_long",
		Encoding:              speechpb.RecognitionConfig_OGG_OPUS,
		SampleRateHertz:       48000,
		AudioChannelCount:     2,
		EnableWordTimeOffsets: true,
		EnableWordConfidence:  true,
		LanguageCode:          "en-US",
	}
}

// RecognizeAudio performs recognition on audio data using the REST API
func (s *Service) RecognizeAudio(audioData []byte) (*TranscriptionResult, error) {
	config := s.createRecognitionConfig()

	audio := &speechpb.RecognitionAudio{
		AudioSource: &speechpb.RecognitionAudio_Content{
			Content: audioData,
		},
	}

	request := &speechpb.RecognizeRequest{
		Config: config,
		Audio:  audio,
	}

	if s.debug {
		log.Printf("Sending %d bytes of audio data to Google Speech REST API", len(audioData))
	}

	response, err := s.client.Recognize(s.ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to recognize audio: %w", err)
	}

	if s.debug {
		log.Printf("Received response with %d results", len(response.Results))
	}

	// Process the first result if available
	if len(response.Results) > 0 && len(response.Results[0].Alternatives) > 0 {
		result := response.Results[0]
		alt := result.Alternatives[0]

		transcriptionResult := &TranscriptionResult{
			Transcript:  alt.Transcript,
			Confidence:  alt.Confidence,
			IsFinal:     true, // REST API results are always final
			WordDetails: alt.Words,
			Language:    result.LanguageCode,
		}

		if s.debug {
			log.Printf("Transcription: %s (confidence: %.2f)", transcriptionResult.Transcript, transcriptionResult.Confidence)
		}

		return transcriptionResult, nil
	}

	return nil, fmt.Errorf("no transcription results received")
}

// Close closes the speech service
func (s *Service) Close() error {
	s.cancel()
	return s.client.Close()
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
