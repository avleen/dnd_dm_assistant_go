package audio

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
)

// New creates a new audio processor
func New(debug bool) *Processor {
	processor := &Processor{
		debug:        debug,
		isProcessing: false,
		oggFiles:     make(map[uint32]*oggwriter.OggWriter),
		// Initialize debug counters
		packetsReceived:   0,
		silenceDetections: 0,
		audioSegments:     0,
		totalBytesWritten: 0,
	}

	if debug {
		log.Printf("[AUDIO] Created new audio processor")
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
	opusPacketDurationMs = 20 // Each Opus packet is typically 20ms

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

	// OGG files for each user (keyed by SSRC)
	oggFiles map[uint32]*oggwriter.OggWriter

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

	// Initialize OGG files map
	p.oggFiles = make(map[uint32]*oggwriter.OggWriter)

	log.Printf("[AUDIO] ✅ Starting audio capture with OGG files per user")
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
	if !exists {
		// Create new OGG file for this user
		filename := fmt.Sprintf("discord_audio_%d_%d.ogg", time.Now().Unix(), packet.SSRC)
		var err error
		oggFile, err = oggwriter.New(filename, discordSampleRate, discordChannels)
		if err != nil {
			log.Printf("[AUDIO] ⚠️ Failed to create OGG file for SSRC %d: %v", packet.SSRC, err)
			return
		}
		p.oggFiles[packet.SSRC] = oggFile
		log.Printf("[AUDIO] 📁 Created OGG file for SSRC %d: %s", packet.SSRC, filename)
	}

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

	if p.debug {
		log.Printf("[AUDIO] 🔇 Silence detected (total: %d silence events)", p.silenceDetections)
	}

	// With the streaming approach, we don't need to process buffered audio
	// The audio is already being streamed to opusdec.exe in real-time
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
