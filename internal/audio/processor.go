package audio

import (
	"bytes"
	"fmt"
	"log"
	"sync"

	"github.com/bwmarrin/discordgo"
)

const (
	// Discord audio constants
	discordSilencePacketSize = 3
	discordSilenceMarker1    = 248
	discordSilenceMarker2    = 255
	discordSilenceMarker3    = 254

	// Audio processing constants
	minAudioDurationSeconds = 0.5
	opusPacketDurationMs    = 20 // Each Opus packet is typically 20ms
)

// Processor handles audio processing from Discord voice channels
type Processor struct {
	debug        bool
	isProcessing bool
	mutex        sync.RWMutex

	// Voice connection
	voiceConnection *discordgo.VoiceConnection

	// Audio buffer for raw PCM data
	audioBuffer *bytes.Buffer
}

// New creates a new audio processor
func New(debug bool) *Processor {
	return &Processor{
		debug:        debug,
		isProcessing: false,
		audioBuffer:  new(bytes.Buffer),
	}
}

// IsProcessing returns whether audio processing is active
func (p *Processor) IsProcessing() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.isProcessing
}

// StartProcessing starts processing audio from the voice connection
func (p *Processor) StartProcessing(vc *discordgo.VoiceConnection) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.isProcessing {
		return fmt.Errorf("audio processing already started")
	}

	p.voiceConnection = vc
	p.isProcessing = true

	// Start processing audio packets in a goroutine
	go p.processAudioPackets()

	return nil
}

// StopProcessing stops audio processing
func (p *Processor) StopProcessing() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.isProcessing {
		return
	}

	p.isProcessing = false
	p.voiceConnection = nil
	p.audioBuffer.Reset()

	if p.debug {
		log.Printf("Stopped audio processing")
	}
}

// processAudioPackets processes incoming audio packets
func (p *Processor) processAudioPackets() {
	if p.debug {
		log.Printf("Started audio packet processing")
	}

	for {
		p.mutex.RLock()
		if !p.isProcessing || p.voiceConnection == nil {
			p.mutex.RUnlock()
			break
		}
		vc := p.voiceConnection
		p.mutex.RUnlock()

		// Receive audio packets
		select {
		case packet, ok := <-vc.OpusRecv:
			if !ok {
				if p.debug {
					log.Printf("Audio channel closed")
				}
				return
			}

			p.processAudioPacket(packet)

		default:
			// Continue listening
		}
	}

	if p.debug {
		log.Printf("Audio packet processing stopped")
	}
}

// processAudioPacket processes a single audio packet
func (p *Processor) processAudioPacket(packet *discordgo.Packet) {
	if packet == nil || len(packet.Opus) == 0 {
		return
	}

	// Log audio reception for debugging
	if p.debug {
		log.Printf("Received audio packet from SSRC %d, size: %d bytes", packet.SSRC, len(packet.Opus))
	}

	// Store the raw opus data for processing
	p.audioBuffer.Write(packet.Opus)

	// Check for Discord silence detection packets
	if p.isSilencePacket(packet) {
		p.handleSilenceDetection()
	}
}

// isSilencePacket checks if the packet indicates silence
func (p *Processor) isSilencePacket(packet *discordgo.Packet) bool {
	return len(packet.Opus) == discordSilencePacketSize &&
		packet.Opus[0] == discordSilenceMarker1 &&
		packet.Opus[1] == discordSilenceMarker2 &&
		packet.Opus[2] == discordSilenceMarker3
}

// handleSilenceDetection processes accumulated audio when silence is detected
func (p *Processor) handleSilenceDetection() {
	if p.debug {
		log.Printf("Silence detected")
	}

	// Calculate approximate duration (each packet is ~20ms)
	estimatedPackets := p.audioBuffer.Len() / 100 // Rough bytes per packet
	estimatedDuration := float32(estimatedPackets) * float32(opusPacketDurationMs) / 1000.0

	if p.debug {
		log.Printf("Audio buffer contains approximately %.2f seconds of audio", estimatedDuration)
	}

	// Process audio if we have sufficient duration
	if estimatedDuration >= minAudioDurationSeconds {
		p.processAudioBuffer()
	}

	// Reset buffer for next audio segment
	p.audioBuffer.Reset()
}

// processAudioBuffer processes the accumulated audio buffer
// This method can be extended to integrate speech-to-text services
func (p *Processor) processAudioBuffer() {
	if p.debug {
		log.Printf("Processing audio buffer with %d bytes of Opus data", p.audioBuffer.Len())
	}

	// Current implementation stores raw Opus data
	// Future enhancements could include:
	// - Opus to PCM decoding
	// - Speech-to-text processing
	// - D&D content analysis
	// - AI-powered DM suggestions
}
