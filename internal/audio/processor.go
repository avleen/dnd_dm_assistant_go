package audio

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"dnd_dm_assistant_go/internal/speech"

	"github.com/bwmarrin/discordgo"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
)

// New creates a new audio processor
func New(debug bool, speechService *speech.Service) *Processor {
	processor := &Processor{
		debug:          debug,
		speechService:  speechService,
		isProcessing:   false,
		oggFiles:       make(map[uint32]*oggwriter.OggWriter),
		oggBuffers:     make(map[uint32]*bytes.Buffer),
		lastPacketTime: make(map[uint32]time.Time),
		// Initialize debug counters
		packetsReceived:   0,
		silenceDetections: 0,
		audioSegments:     0,
		totalBytesWritten: 0,
	}

	if debug {
		log.Printf("[AUDIO] Created new audio processor")
		if speechService != nil {
			log.Printf("[AUDIO] Speech-to-text service available")
		} else {
			log.Printf("[AUDIO] Speech-to-text service disabled")
		}
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
	opusPacketDurationMs = 20              // Each Opus packet is typically 20ms
	silenceThreshold     = 2 * time.Second // Send to Google after 2 seconds of silence

	// Discord audio format
	discordSampleRate = 48000
	discordChannels   = 2
	discordFrameSize  = 960 // 20ms at 48kHz
)

// Processor handles audio processing from Discord voice channels
type Processor struct {
	debug         bool
	speechService *speech.Service
	isProcessing  bool
	mutex         sync.RWMutex

	// Voice connection
	voiceConnection *discordgo.VoiceConnection

	// OGG files for each user (keyed by SSRC)
	oggFiles map[uint32]*oggwriter.OggWriter

	// OGG buffers for each user (keyed by SSRC)
	oggBuffers map[uint32]*bytes.Buffer

	// Last packet time for each user (keyed by SSRC) - for silence detection
	lastPacketTime map[uint32]time.Time

	// Debug counters
	packetsReceived   int64
	silenceDetections int64
	audioSegments     int64
	totalBytesWritten int64
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
	p.totalBytesWritten = 0

	// Initialize maps
	p.oggFiles = make(map[uint32]*oggwriter.OggWriter)
	p.oggBuffers = make(map[uint32]*bytes.Buffer)
	p.lastPacketTime = make(map[uint32]time.Time)

	log.Printf("[AUDIO] ✅ Starting audio capture with OGG files per user")
	if p.debug {
		log.Printf("[AUDIO] Voice connection guild: %s, channel: %s", vc.GuildID, vc.ChannelID)
		log.Printf("[AUDIO] Audio format: %dHz, %d channels, %dms packets",
			discordSampleRate, discordChannels, opusPacketDurationMs)
	}

	// Start processing audio packets in a goroutine
	go p.processAudioPackets()

	// Start background silence detector
	go p.silenceDetector()

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

	// Send any remaining buffered audio to Google before closing
	if p.speechService != nil {
		for ssrc := range p.oggBuffers {
			p.sendBufferToGoogle(ssrc)
		}
	}

	// Close all OGG files
	for ssrc, oggFile := range p.oggFiles {
		if oggFile != nil {
			err := oggFile.Close()
			if err != nil {
				log.Printf("[AUDIO] ⚠️ Failed to close OGG file for SSRC %d: %v", ssrc, err)
			} else {
				log.Printf("[AUDIO] 📁 Closed OGG file for SSRC %d", ssrc)
			}
		}
	}
	p.oggFiles = make(map[uint32]*oggwriter.OggWriter)

	// Clear other maps
	p.oggBuffers = make(map[uint32]*bytes.Buffer)
	p.lastPacketTime = make(map[uint32]time.Time)

	log.Printf("[AUDIO] ⏹️ Stopped audio processing")
	if p.debug {
		log.Printf("[AUDIO] Final stats: %d packets, %d silence detections, %d audio segments",
			p.packetsReceived, p.silenceDetections, p.audioSegments)
		log.Printf("[AUDIO] Total bytes written: %d", p.totalBytesWritten)
	}
}

// processAudioPacket processes a single audio packet
func (p *Processor) processAudioPacket(packet *discordgo.Packet) {
	if packet == nil || len(packet.Opus) == 0 {
		return
	}

	// Update counters
	p.packetsReceived++

	// Check for Discord silence detection packets
	isSilence := p.isSilencePacket(packet)
	if isSilence {
		p.handleSilenceDetection()
		// Skip saving silence packets to OGG files
		return
	}

	// Get or create OGG writer for this SSRC (user)
	oggFile, exists := p.oggFiles[packet.SSRC]
	var oggBuffer *bytes.Buffer

	if !exists {
		var err error
		// Create a buffer to hold OGG data in memory
		oggBuffer = &bytes.Buffer{}
		oggFile, err = oggwriter.NewWith(oggBuffer, discordSampleRate, discordChannels)
		if err != nil {
			log.Printf("[AUDIO] ⚠️ Failed to create OGG buffer for SSRC %d: %v", packet.SSRC, err)
			return
		}
		p.oggFiles[packet.SSRC] = oggFile
		p.oggBuffers[packet.SSRC] = oggBuffer
		log.Printf("[AUDIO] 📁 Created OGG buffer for SSRC %d", packet.SSRC)
	} else {
		// Buffer already exists, no need to get reference
	}

	// Update last packet time for this SSRC
	p.lastPacketTime[packet.SSRC] = time.Now()

	// Create RTP packet from Discord packet
	rtpPacket := &rtp.Packet{
		Header: rtp.Header{
			Version:        2,
			Padding:        false,
			Extension:      false,
			Marker:         false,
			PayloadType:    111, // Opus payload type
			SequenceNumber: packet.Sequence,
			Timestamp:      packet.Timestamp,
			SSRC:           packet.SSRC,
		},
		Payload: packet.Opus,
	}

	// Write RTP packet to OGG file
	err := oggFile.WriteRTP(rtpPacket)
	if err != nil {
		log.Printf("[AUDIO] ⚠️ Failed to write RTP packet to OGG file for SSRC %d: %v", packet.SSRC, err)
	} else {
		p.totalBytesWritten += int64(len(packet.Opus))
	}

	// Log packet info
	if p.debug {
		if p.packetsReceived%20 == 0 { // Log every 20 audio packets
			log.Printf("[AUDIO] 📤 Audio packet #%d from SSRC %d (%d bytes)",
				p.packetsReceived, packet.SSRC, len(packet.Opus))
		}
	}

	// Every 50 packets (1 second), log status
	if p.debug && p.packetsReceived%50 == 0 {
		estimatedDuration := float32(p.packetsReceived) * float32(opusPacketDurationMs) / 1000.0
		log.Printf("[AUDIO] 📊 Captured: %d packets processed, ~%.1fs total (%d bytes saved)",
			p.packetsReceived, estimatedDuration, p.totalBytesWritten)
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
}

// writeDebugFile writes the OGG buffer to disk for manual testing
func (p *Processor) writeDebugFile(ssrc uint32, data []byte) {
	if len(data) == 0 {
		return
	}

	// Create filename with timestamp and SSRC
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("debug_audio_%s_%d.ogg", timestamp, ssrc)

	if err := os.WriteFile(filename, data, 0644); err != nil {
		if p.debug {
			log.Printf("[AUDIO] ⚠️ Failed to write debug file %s: %v", filename, err)
		}
	} else {
		if p.debug {
			log.Printf("[AUDIO] 📁 Wrote debug file %s (%d bytes)", filename, len(data))
		}
	}
}

// sendBufferToGoogle sends the accumulated OGG buffer to Google for transcription
func (p *Processor) sendBufferToGoogle(ssrc uint32) {
	if p.speechService == nil {
		return
	}

	buffer, exists := p.oggBuffers[ssrc]
	if !exists || buffer.Len() == 0 {
		return
	}

	// Send the buffered OGG data to Google using REST API
	result, err := p.speechService.RecognizeAudio(buffer.Bytes())
	if err != nil {
		if p.debug {
			log.Printf("[AUDIO] ⚠️ Failed to send buffered audio to Google for SSRC %d: %v", ssrc, err)
		}

		// Write the failed buffer to disk for manual testing
		p.writeDebugFile(ssrc, buffer.Bytes())
	} else {
		if p.debug {
			log.Printf("[AUDIO] � Sent %d bytes of buffered OGG audio to Google for SSRC %d", buffer.Len(), ssrc)
		}

		// Print the transcription result to stdout
		if result != nil {
			fmt.Printf("[TRANSCRIPTION] SSRC %d [FINAL]: %s (confidence: %.2f)\n",
				ssrc, result.Transcript, result.Confidence)

			// Also log to internal logging if debug is enabled
			if p.debug {
				log.Printf("[AUDIO] 📝 Transcription for SSRC %d [FINAL]: %s (confidence: %.2f)",
					ssrc, result.Transcript, result.Confidence)
			}
		}
	}

	// Flush the buffer after sending
	buffer.Reset()

	// Update last packet time to prevent immediate re-sending
	p.lastPacketTime[ssrc] = time.Now()
}

// processAudioPackets processes incoming audio packets
func (p *Processor) processAudioPackets() {
	if p.voiceConnection == nil {
		log.Printf("[AUDIO] ❌ No voice connection available")
		return
	}

	log.Printf("[AUDIO] 🎧 Started listening for Discord audio packets...")
	if p.debug {
		log.Printf("[AUDIO] Voice connection ready: %v", p.voiceConnection.Ready)
		log.Printf("[AUDIO] OpusRecv channel: %p", p.voiceConnection.OpusRecv)
	}

	// Listen for packets from Discord's OpusRecv channel
	for packet := range p.voiceConnection.OpusRecv {
		if !p.isProcessing {
			log.Printf("[AUDIO] 🛑 Audio processing stopped, exiting packet loop")
			return
		}

		if packet != nil {
			p.processAudioPacket(packet)
		}
	}

	log.Printf("[AUDIO] 🎧 Finished processing audio packets")
}

// silenceDetector runs in background checking for silence every 100ms
func (p *Processor) silenceDetector() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	log.Printf("[AUDIO] 🔍 Started background silence detector (checking every 100ms)")

	for range ticker.C {
		if !p.isProcessing {
			log.Printf("[AUDIO] 🔍 Background silence detector stopped")
			return
		}
		p.checkAllForSilence()
	}
}

// checkAllForSilence checks all SSRCs for silence and sends buffers if needed
func (p *Processor) checkAllForSilence() {
	if p.speechService == nil {
		return
	}

	now := time.Now()

	// Check each SSRC for silence
	for ssrc, lastTime := range p.lastPacketTime {
		if now.Sub(lastTime) > silenceThreshold {
			// Check if this SSRC has buffered audio to send
			if buffer, exists := p.oggBuffers[ssrc]; exists && buffer.Len() > 0 {
				if p.debug {
					log.Printf("[AUDIO] 🔍 Detected silence for SSRC %d (%.2fs), sending %d bytes to Google",
						ssrc, now.Sub(lastTime).Seconds(), buffer.Len())
				}
				p.sendBufferToGoogle(ssrc)
			}
		}
	}
}
