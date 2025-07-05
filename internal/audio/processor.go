package audio

import (
	"bytes"
	"fmt"
	"log"
	"sync"

	"github.com/bwmarrin/discordgo"
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

	// For now, just log that we received audio
	if p.debug {
		log.Printf("Received audio packet from SSRC %d, size: %d bytes", packet.SSRC, len(packet.Opus))
	}

	// TODO: Decode Opus to PCM when opus decoder is available
	// For now, just store the raw opus data
	p.audioBuffer.Write(packet.Opus)

	// Calculate approximate duration based on packet size (rough estimation)
	// Each opus packet is typically 20ms of audio
	packetCount := p.audioBuffer.Len() / 100 // rough estimation
	duration := float32(packetCount) * 0.02  // 20ms per packet

	if p.debug && duration > 1.0 { // Log every second of audio
		log.Printf("Audio buffer duration: approximately %.2f seconds", duration)
	}

	// Check for silence detection (Discord sends specific silence packets)
	if len(packet.Opus) == 3 && packet.Opus[0] == 248 && packet.Opus[1] == 255 && packet.Opus[2] == 254 {
		if p.debug {
			log.Printf("Silence detected, buffer has approximately %.2f seconds of audio", duration)
		}

		// Here you would typically process the accumulated audio
		// For now, we'll just reset the buffer
		if duration > 0.5 { // Only process if we have at least 500ms of audio
			p.processAudioBuffer()
		}

		p.audioBuffer.Reset()
	}
}

// processAudioBuffer processes the accumulated audio buffer
func (p *Processor) processAudioBuffer() {
	if p.debug {
		log.Printf("Processing audio buffer with %d bytes", p.audioBuffer.Len())
	}

	// TODO: This is where you would add speech-to-text processing
	// For now, we just acknowledge that we have audio to process

	// Future implementations could:
	// 1. Send audio to a speech-to-text service
	// 2. Process the transcription for D&D-related content
	// 3. Generate AI-powered DM suggestions
	// 4. Send helpful messages back to Discord
}
