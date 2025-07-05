package audio

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pion/opus"
)

// New creates a new audio processor
func New(debug bool) *Processor {
	// Create Opus decoder for Discord audio (48kHz, 2 channels)
	decoder := opus.NewDecoder()

	processor := &Processor{
		debug:        debug,
		isProcessing: false,
		audioBuffer:  new(bytes.Buffer),
		opusDecoder:  decoder,
		// Initialize debug counters
		packetsReceived:   0,
		silenceDetections: 0,
		audioSegments:     0,
		totalBytesOpus:    0,
		totalBytesPCM:     0,
	}

	if debug {
		log.Printf("[AUDIO] Created new audio processor with debug logging enabled")
	}

	return processor
}

const (
	// Discord audio constants
	discordSilencePacketSize = 3
	discordSilenceMarker1    = 248
	discordSilenceMarker2    = 255
	discordSilenceMarker3    = 254

	// Audio processing constants
	minAudioDurationSeconds = 0.5
	opusPacketDurationMs    = 20 // Each Opus packet is typically 20ms

	// Discord audio format
	discordSampleRate = 48000
	discordChannels   = 2
	discordFrameSize  = 960 // 20ms at 48kHz
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

	// Opus decoder
	opusDecoder opus.Decoder

	// Debug counters
	packetsReceived   int64
	silenceDetections int64
	audioSegments     int64
	totalBytesOpus    int64
	totalBytesPCM     int64
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

	// Reset debug counters
	p.packetsReceived = 0
	p.silenceDetections = 0
	p.audioSegments = 0
	p.totalBytesOpus = 0
	p.totalBytesPCM = 0

	log.Printf("[AUDIO] ✅ Starting audio processing stream")
	if p.debug {
		log.Printf("[AUDIO] Voice connection guild: %s, channel: %s", vc.GuildID, vc.ChannelID)
		log.Printf("[AUDIO] Audio format: %dHz, %d channels, %dms packets",
			discordSampleRate, discordChannels, opusPacketDurationMs)
	}

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

	// Reset audio buffer
	p.audioBuffer.Reset()

	log.Printf("[AUDIO] ⏹️ Stopped audio processing stream")
	if p.debug {
		log.Printf("[AUDIO] Final stats: %d packets, %d silence detections, %d audio segments",
			p.packetsReceived, p.silenceDetections, p.audioSegments)
		log.Printf("[AUDIO] Data processed: %d bytes Opus → %d bytes PCM",
			p.totalBytesOpus, p.totalBytesPCM)
	}
}

// processAudioPackets processes incoming audio packets
func (p *Processor) processAudioPackets() {
	if p.voiceConnection == nil {
		log.Printf("[AUDIO] ❌ No voice connection available")
		return
	}

	log.Printf("[AUDIO] 🎧 Started audio packet processing loop")
	if p.debug {
		log.Printf("[AUDIO] Listening for Opus packets on voice connection...")
		log.Printf("[AUDIO] Voice connection ready: %v", p.voiceConnection.Ready)
		log.Printf("[AUDIO] OpusRecv channel: %p", p.voiceConnection.OpusRecv)
	}

	// Set up a ticker to log status every 10 seconds if no audio is received
	packetCount := int64(0)
	lastStatusTime := time.Now()

	for {
		select {
		case packet := <-p.voiceConnection.OpusRecv:
			if !p.isProcessing {
				log.Printf("[AUDIO] 🛑 Audio processing stopped, exiting packet loop")
				return
			}

			packetCount++
			if packetCount == 1 {
				log.Printf("[AUDIO] 🎉 First audio packet received!")
			}

			p.processAudioPacket(packet)

		default:
			// Check if we should continue processing
			if !p.isProcessing {
				log.Printf("[AUDIO] 🛑 Audio processing stopped, exiting packet loop")
				return
			}

			// Log status every 10 seconds if no packets received
			if time.Since(lastStatusTime) > 10*time.Second {
				if packetCount == 0 {
					log.Printf("[AUDIO] ⏳ Still waiting for audio packets... (connection ready: %v)",
						p.voiceConnection.Ready)
				}
				lastStatusTime = time.Now()
			}

			// Brief sleep to prevent busy waiting
			time.Sleep(1 * time.Millisecond)
		}
	}
}

// processAudioPacket processes a single audio packet
func (p *Processor) processAudioPacket(packet *discordgo.Packet) {
	if packet == nil || len(packet.Opus) == 0 {
		return
	}

	// Update counters
	p.packetsReceived++

	// Check for Discord silence detection packets first
	if p.isSilencePacket(packet) {
		p.handleSilenceDetection()
		return
	}

	// Log audio reception for debugging
	if p.debug {
		log.Printf("[AUDIO] 📦 Packet #%d from SSRC %d: %d bytes",
			p.packetsReceived, packet.SSRC, len(packet.Opus))
	}

	// Store the raw opus data for processing
	p.audioBuffer.Write(packet.Opus)
	p.totalBytesOpus += int64(len(packet.Opus))

	// Every 50 packets (1 second), log buffer status
	if p.debug && p.packetsReceived%50 == 0 {
		estimatedDuration := float32(p.packetsReceived) * float32(opusPacketDurationMs) / 1000.0
		log.Printf("[AUDIO] 📊 Buffer status: %d bytes, ~%.1fs recorded",
			p.audioBuffer.Len(), estimatedDuration)
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
	p.silenceDetections++

	log.Printf("[AUDIO] 🔇 Silence detection #%d - processing audio segment", p.silenceDetections)

	// Calculate approximate duration (each packet is ~20ms)
	estimatedPackets := p.audioBuffer.Len() / 100 // Rough bytes per packet
	estimatedDuration := float32(estimatedPackets) * float32(opusPacketDurationMs) / 1000.0

	if p.debug {
		log.Printf("[AUDIO] Audio segment contains ~%d packets (%.2f seconds, %d bytes)",
			estimatedPackets, estimatedDuration, p.audioBuffer.Len())
	}

	// Process audio if we have sufficient duration
	if estimatedDuration >= minAudioDurationSeconds {
		log.Printf("[AUDIO] ✅ Processing audio segment (>%.1fs threshold)", minAudioDurationSeconds)
		p.processAudioBuffer()
	} else if p.debug {
		log.Printf("[AUDIO] ⏭️ Skipping short audio segment (%.2fs < %.1fs threshold)",
			estimatedDuration, minAudioDurationSeconds)
	}

	// Reset buffer for next audio segment
	p.audioBuffer.Reset()
}

// processAudioBuffer processes the accumulated audio buffer
func (p *Processor) processAudioBuffer() {
	p.audioSegments++

	log.Printf("[AUDIO] 🎵 Processing audio segment #%d (%d bytes Opus)",
		p.audioSegments, p.audioBuffer.Len())

	// Convert Opus to PCM if we have audio data
	if p.audioBuffer.Len() > 0 {
		opusData := p.audioBuffer.Bytes()

		// Decode Opus to PCM
		// The Pion Opus decoder expects a proper buffer for output
		pcmBuffer := make([]byte, len(opusData)*4) // Estimate PCM size
		bandwidth, isStereo, err := p.opusDecoder.Decode(opusData, pcmBuffer)
		if err != nil {
			log.Printf("[AUDIO] ❌ Failed to decode Opus audio: %v", err)
			return
		}

		p.totalBytesPCM += int64(len(pcmBuffer))

		log.Printf("[AUDIO] ✅ Segment #%d decoded: %d bytes Opus → %d bytes PCM (bandwidth: %v, stereo: %v)",
			p.audioSegments, len(opusData), len(pcmBuffer), bandwidth, isStereo)

		// TODO: Here you can send pcmBuffer to speech recognition service
		// For now, we just log that we have processed audio
		if p.debug {
			log.Printf("[AUDIO] 📤 Ready to send %d bytes PCM to speech service", len(pcmBuffer))
		}
	}
}
