package analysis

import (
	"math"
	"sync"
)

// PeakMeter measures peak signal levels
type PeakMeter struct {
	peak       float64
	hold       float64
	holdTime   float64
	decayRate  float64
	sampleRate float64
	holdCount  int
	mu         sync.Mutex
}

// NewPeakMeter creates a new peak meter
func NewPeakMeter(sampleRate float64) *PeakMeter {
	return &PeakMeter{
		sampleRate: sampleRate,
		holdTime:   3.0,  // 3 seconds default
		decayRate:  20.0, // 20 dB/second
	}
}

// SetHoldTime sets the peak hold time in seconds
func (pm *PeakMeter) SetHoldTime(seconds float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.holdTime = seconds
}

// SetDecayRate sets the peak decay rate in dB/second
func (pm *PeakMeter) SetDecayRate(dbPerSecond float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.decayRate = dbPerSecond
}

// Process updates the peak meter with new samples
func (pm *PeakMeter) Process(samples []float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// Find peak in current block
	blockPeak := 0.0
	for _, sample := range samples {
		absSample := math.Abs(sample)
		if absSample > blockPeak {
			blockPeak = absSample
		}
	}
	
	// Update peak with decay
	samplesPerSecond := pm.sampleRate
	decayPerSample := pm.decayRate / samplesPerSecond / 20.0 * math.Log(10) // Convert dB to linear
	pm.peak *= math.Exp(-decayPerSample * float64(len(samples)))
	
	// Update peak if new value is higher
	if blockPeak > pm.peak {
		pm.peak = blockPeak
	}
	
	// Update hold
	if blockPeak > pm.hold {
		pm.hold = blockPeak
		pm.holdCount = int(pm.holdTime * pm.sampleRate)
	} else {
		pm.holdCount -= len(samples)
		if pm.holdCount <= 0 {
			pm.hold = pm.peak
			pm.holdCount = 0
		}
	}
}

// GetPeak returns the current peak level (linear)
func (pm *PeakMeter) GetPeak() float64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.peak
}

// GetPeakDB returns the current peak level in decibels
func (pm *PeakMeter) GetPeakDB() float64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.peak > 0 {
		return 20.0 * math.Log10(pm.peak)
	}
	return -math.Inf(1)
}

// GetHold returns the held peak level (linear)
func (pm *PeakMeter) GetHold() float64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.hold
}

// GetHoldDB returns the held peak level in decibels
func (pm *PeakMeter) GetHoldDB() float64 {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.hold > 0 {
		return 20.0 * math.Log10(pm.hold)
	}
	return -math.Inf(1)
}

// Reset clears the peak and hold values
func (pm *PeakMeter) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.peak = 0
	pm.hold = 0
	pm.holdCount = 0
}

// RMSMeter measures RMS (Root Mean Square) levels
type RMSMeter struct {
	windowSize int
	buffer     []float64
	writePos   int
	sum        float64
	count      int
	mu         sync.Mutex
}

// NewRMSMeter creates a new RMS meter with specified window size
func NewRMSMeter(windowSizeSamples int) *RMSMeter {
	return &RMSMeter{
		windowSize: windowSizeSamples,
		buffer:     make([]float64, windowSizeSamples),
	}
}

// Process updates the RMS meter with new samples
func (rm *RMSMeter) Process(samples []float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	for _, sample := range samples {
		// Remove old value from sum
		oldValue := rm.buffer[rm.writePos]
		rm.sum -= oldValue * oldValue
		
		// Add new value
		rm.buffer[rm.writePos] = sample
		rm.sum += sample * sample
		
		// Update position
		rm.writePos = (rm.writePos + 1) % rm.windowSize
		if rm.count < rm.windowSize {
			rm.count++
		}
	}
}

// GetRMS returns the current RMS level (linear)
func (rm *RMSMeter) GetRMS() float64 {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if rm.count == 0 {
		return 0
	}
	
	return math.Sqrt(rm.sum / float64(rm.count))
}

// GetRMSDB returns the current RMS level in decibels
func (rm *RMSMeter) GetRMSDB() float64 {
	rms := rm.GetRMS()
	if rms > 0 {
		return 20.0 * math.Log10(rms)
	}
	return -math.Inf(1)
}

// Reset clears the RMS buffer
func (rm *RMSMeter) Reset() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	for i := range rm.buffer {
		rm.buffer[i] = 0
	}
	rm.sum = 0
	rm.count = 0
	rm.writePos = 0
}

// LUFSMeter implements ITU-R BS.1770-4 loudness measurement
type LUFSMeter struct {
	sampleRate   float64
	channels     int
	momentary    *LUFSBlock
	shortTerm    *LUFSBlock
	integrated   *LUFSIntegrated
	preFilter    [][]*BiquadFilter // K-weighting pre-filter (per channel)
	highShelf    [][]*BiquadFilter // K-weighting high shelf (per channel)
	channelPower []float64
	mu           sync.Mutex
}

// LUFSBlock represents a gated loudness measurement block
type LUFSBlock struct {
	blockSize    int
	overlap      int
	blocks       [][]float64
	writePos     int
	gatingThresh float64
	buffer       []float64
	bufferPos    int
}

// LUFSIntegrated handles integrated loudness measurement
type LUFSIntegrated struct {
	blocks       []float64
	absoluteGate float64
	relativeGate float64
}

// BiquadFilter for K-weighting
type BiquadFilter struct {
	b0, b1, b2 float64
	a1, a2     float64
	x1, x2     float64
	y1, y2     float64
}

// NewLUFSMeter creates a new LUFS meter
func NewLUFSMeter(sampleRate float64, channels int) *LUFSMeter {
	lm := &LUFSMeter{
		sampleRate:   sampleRate,
		channels:     channels,
		channelPower: make([]float64, channels),
		preFilter:    make([][]*BiquadFilter, channels),
		highShelf:    make([][]*BiquadFilter, channels),
	}
	
	// Initialize K-weighting filters for each channel
	for ch := 0; ch < channels; ch++ {
		// Pre-filter (high-pass)
		lm.preFilter[ch] = []*BiquadFilter{lm.createKWeightingPreFilter()}
		
		// High shelf filter
		lm.highShelf[ch] = []*BiquadFilter{lm.createKWeightingHighShelf()}
	}
	
	// Momentary loudness: 400ms window, 100ms update
	momentarySize := int(0.4 * sampleRate)
	momentaryOverlap := int(0.3 * sampleRate)
	lm.momentary = &LUFSBlock{
		blockSize:    momentarySize,
		overlap:      momentaryOverlap,
		blocks:       make([][]float64, 1),
		buffer:       make([]float64, momentarySize*channels),
		gatingThresh: -math.Inf(1), // No gating for momentary
	}
	lm.momentary.blocks[0] = make([]float64, channels)
	
	// Short-term loudness: 3s window, 100ms update
	shortTermSize := int(3.0 * sampleRate)
	shortTermOverlap := int(2.9 * sampleRate)
	lm.shortTerm = &LUFSBlock{
		blockSize:    shortTermSize,
		overlap:      shortTermOverlap,
		blocks:       make([][]float64, 30), // Store 30 blocks for 3s
		buffer:       make([]float64, shortTermSize*channels),
		gatingThresh: -math.Inf(1), // No gating for short-term
	}
	for i := range lm.shortTerm.blocks {
		lm.shortTerm.blocks[i] = make([]float64, channels)
	}
	
	// Integrated loudness
	lm.integrated = &LUFSIntegrated{
		blocks:       make([]float64, 0, 1000), // Pre-allocate for efficiency
		absoluteGate: -70.0,                    // LUFS
		relativeGate: -10.0,                    // dB below ungated loudness
	}
	
	return lm
}

// createKWeightingPreFilter creates the K-weighting pre-filter (high-pass)
func (lm *LUFSMeter) createKWeightingPreFilter() *BiquadFilter {
	// ITU-R BS.1770-4 pre-filter coefficients
	f0 := 1681.974450955533
	G := 3.999843853973347
	Q := 0.7071752369554196
	K := math.Tan(math.Pi * f0 / lm.sampleRate)
	Vh := math.Pow(10.0, G/20.0)
	Vb := math.Pow(Vh, 0.4996667741545416)
	
	a0 := 1.0 + K/Q + K*K
	
	return &BiquadFilter{
		b0: (Vh + Vb*K/Q + K*K) / a0,
		b1: 2.0 * (K*K - Vh) / a0,
		b2: (Vh - Vb*K/Q + K*K) / a0,
		a1: 2.0 * (K*K - 1.0) / a0,
		a2: (1.0 - K/Q + K*K) / a0,
	}
}

// createKWeightingHighShelf creates the K-weighting high shelf filter
func (lm *LUFSMeter) createKWeightingHighShelf() *BiquadFilter {
	// ITU-R BS.1770-4 high shelf coefficients
	f0 := 38.13547087602444
	Q := 0.5003270373238773
	K := math.Tan(math.Pi * f0 / lm.sampleRate)
	
	a0 := 1.0 + K/Q + K*K
	
	return &BiquadFilter{
		b0: (1.0 + math.Sqrt(2.0)*K + K*K) / a0,
		b1: 2.0 * (K*K - 1.0) / a0,
		b2: (1.0 - math.Sqrt(2.0)*K + K*K) / a0,
		a1: 2.0 * (K*K - 1.0) / a0,
		a2: (1.0 - K/Q + K*K) / a0,
	}
}

// processBiquad processes a sample through the biquad filter
func (bf *BiquadFilter) processBiquad(input float64) float64 {
	output := bf.b0*input + bf.b1*bf.x1 + bf.b2*bf.x2 - bf.a1*bf.y1 - bf.a2*bf.y2
	
	bf.x2 = bf.x1
	bf.x1 = input
	bf.y2 = bf.y1
	bf.y1 = output
	
	return output
}

// Process updates the LUFS meter with new multichannel samples
// samples should be interleaved: [ch0, ch1, ch0, ch1, ...]
func (lm *LUFSMeter) Process(samples []float64) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	// Process samples through K-weighting filters
	filtered := make([]float64, len(samples))
	for i := 0; i < len(samples); i += lm.channels {
		for ch := 0; ch < lm.channels; ch++ {
			if i+ch < len(samples) {
				// Apply K-weighting filters
				sample := samples[i+ch]
				sample = lm.preFilter[ch][0].processBiquad(sample)
				sample = lm.highShelf[ch][0].processBiquad(sample)
				filtered[i+ch] = sample
			}
		}
	}
	
	// Update momentary and short-term blocks
	lm.updateBlock(lm.momentary, filtered)
	lm.updateBlock(lm.shortTerm, filtered)
	
	// Update integrated measurement (every 100ms of new data)
	samplesFor100ms := int(0.1 * lm.sampleRate * float64(lm.channels))
	if len(filtered) >= samplesFor100ms {
		// Calculate mean square for this block
		meanSquare := 0.0
		for ch := 0; ch < lm.channels; ch++ {
			chPower := 0.0
			count := 0
			for i := ch; i < len(filtered); i += lm.channels {
				chPower += filtered[i] * filtered[i]
				count++
			}
			if count > 0 {
				meanSquare += chPower / float64(count)
			}
		}
		
		if meanSquare > 0 {
			loudness := -0.691 + 10.0*math.Log10(meanSquare)
			lm.integrated.blocks = append(lm.integrated.blocks, loudness)
		}
	}
}

// updateBlock updates a loudness measurement block
func (lm *LUFSMeter) updateBlock(block *LUFSBlock, filtered []float64) {
	// Add samples to buffer
	for _, sample := range filtered {
		block.buffer[block.bufferPos] = sample
		block.bufferPos++
		
		// When buffer is full, calculate loudness
		if block.bufferPos >= len(block.buffer) {
			// Calculate mean square per channel
			for ch := 0; ch < lm.channels; ch++ {
				power := 0.0
				count := 0
				for i := ch; i < len(block.buffer); i += lm.channels {
					power += block.buffer[i] * block.buffer[i]
					count++
				}
				if count > 0 {
					lm.channelPower[ch] = power / float64(count)
				}
			}
			
			// Store channel powers
			copy(block.blocks[block.writePos], lm.channelPower)
			block.writePos = (block.writePos + 1) % len(block.blocks)
			
			// Shift buffer by (blockSize - overlap)
			shift := block.blockSize - block.overlap
			copy(block.buffer, block.buffer[shift*lm.channels:])
			block.bufferPos = block.overlap * lm.channels
		}
	}
}

// GetMomentaryLUFS returns the momentary loudness in LUFS
func (lm *LUFSMeter) GetMomentaryLUFS() float64 {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	return lm.calculateBlockLoudness(lm.momentary)
}

// GetShortTermLUFS returns the short-term loudness in LUFS
func (lm *LUFSMeter) GetShortTermLUFS() float64 {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	return lm.calculateBlockLoudness(lm.shortTerm)
}

// GetIntegratedLUFS returns the integrated loudness in LUFS
func (lm *LUFSMeter) GetIntegratedLUFS() float64 {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	if len(lm.integrated.blocks) == 0 {
		return -math.Inf(1)
	}
	
	// First pass: calculate ungated loudness
	sum := 0.0
	for _, block := range lm.integrated.blocks {
		sum += math.Pow(10.0, block/10.0)
	}
	ungatedLoudness := 10.0 * math.Log10(sum/float64(len(lm.integrated.blocks)))
	
	// Apply absolute gate (-70 LUFS)
	sum = 0.0
	count := 0
	for _, block := range lm.integrated.blocks {
		if block >= lm.integrated.absoluteGate {
			sum += math.Pow(10.0, block/10.0)
			count++
		}
	}
	
	if count == 0 {
		return -math.Inf(1)
	}
	
	// Apply relative gate (10 dB below ungated)
	relativeThreshold := ungatedLoudness + lm.integrated.relativeGate
	sum = 0.0
	count = 0
	for _, block := range lm.integrated.blocks {
		if block >= lm.integrated.absoluteGate && block >= relativeThreshold {
			sum += math.Pow(10.0, block/10.0)
			count++
		}
	}
	
	if count == 0 {
		return -math.Inf(1)
	}
	
	return 10.0 * math.Log10(sum/float64(count))
}

// calculateBlockLoudness calculates loudness for a measurement block
func (lm *LUFSMeter) calculateBlockLoudness(block *LUFSBlock) float64 {
	// Sum channel powers with channel weighting
	totalPower := 0.0
	validBlocks := 0
	
	for _, powers := range block.blocks {
		blockPower := 0.0
		for ch, power := range powers {
			// Apply channel weighting (ITU-R BS.1770-4)
			// For stereo: both channels have weight 1.0
			// For 5.1: L/R = 1.0, C = 1.0, Ls/Rs = 1.41, LFE = 0
			weight := 1.0
			if lm.channels > 2 && ch >= 3 && ch <= 4 {
				weight = 1.41 // Surround channels
			} else if lm.channels > 5 && ch == 3 {
				weight = 0.0 // LFE
			}
			
			blockPower += weight * power
		}
		
		if blockPower > 0 {
			loudness := -0.691 + 10.0*math.Log10(blockPower)
			if loudness >= block.gatingThresh {
				totalPower += math.Pow(10.0, loudness/10.0)
				validBlocks++
			}
		}
	}
	
	if validBlocks == 0 {
		return -math.Inf(1)
	}
	
	return 10.0 * math.Log10(totalPower/float64(validBlocks))
}

// GetLoudnessRange returns the loudness range (LRA) in LU
func (lm *LUFSMeter) GetLoudnessRange() float64 {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	if len(lm.integrated.blocks) < 20 {
		return 0 // Not enough data
	}
	
	// Sort blocks for percentile calculation
	sorted := make([]float64, 0, len(lm.integrated.blocks))
	absGate := lm.integrated.absoluteGate
	
	for _, block := range lm.integrated.blocks {
		if block >= absGate {
			sorted = append(sorted, block)
		}
	}
	
	if len(sorted) < 20 {
		return 0
	}
	
	// Simple bubble sort for percentiles
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	// Calculate 10th and 95th percentiles
	idx10 := int(float64(len(sorted)) * 0.1)
	idx95 := int(float64(len(sorted)) * 0.95)
	
	return sorted[idx95] - sorted[idx10]
}

// Reset clears all measurements
func (lm *LUFSMeter) Reset() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	// Reset buffers
	for i := range lm.momentary.buffer {
		lm.momentary.buffer[i] = 0
	}
	for i := range lm.shortTerm.buffer {
		lm.shortTerm.buffer[i] = 0
	}
	
	// Reset positions
	lm.momentary.bufferPos = 0
	lm.shortTerm.bufferPos = 0
	lm.momentary.writePos = 0
	lm.shortTerm.writePos = 0
	
	// Clear blocks
	for i := range lm.momentary.blocks {
		for j := range lm.momentary.blocks[i] {
			lm.momentary.blocks[i][j] = 0
		}
	}
	for i := range lm.shortTerm.blocks {
		for j := range lm.shortTerm.blocks[i] {
			lm.shortTerm.blocks[i][j] = 0
		}
	}
	
	// Clear integrated
	lm.integrated.blocks = lm.integrated.blocks[:0]
	
	// Reset filters
	for ch := range lm.preFilter {
		for _, filter := range lm.preFilter[ch] {
			filter.x1, filter.x2, filter.y1, filter.y2 = 0, 0, 0, 0
		}
	}
	for ch := range lm.highShelf {
		for _, filter := range lm.highShelf[ch] {
			filter.x1, filter.x2, filter.y1, filter.y2 = 0, 0, 0, 0
		}
	}
}