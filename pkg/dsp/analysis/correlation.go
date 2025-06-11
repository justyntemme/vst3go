package analysis

import (
	"math"
	"sync"
)

// CorrelationMeter measures stereo correlation/phase relationships
type CorrelationMeter struct {
	windowSize   int
	bufferL      []float64
	bufferR      []float64
	writePos     int
	count        int
	correlation  float64
	averaging    float64
	peakHold     float64
	peakHoldTime float64
	peakHoldCount int
	sampleRate   float64
	mu           sync.Mutex
}

// NewCorrelationMeter creates a new correlation meter
func NewCorrelationMeter(windowSizeSamples int, sampleRate float64) *CorrelationMeter {
	return &CorrelationMeter{
		windowSize:   windowSizeSamples,
		bufferL:      make([]float64, windowSizeSamples),
		bufferR:      make([]float64, windowSizeSamples),
		averaging:    0.9,
		peakHoldTime: 3.0, // 3 seconds
		sampleRate:   sampleRate,
		correlation:  0.0, // Initialize to neutral
		peakHold:     1.0, // Initialize to best case
	}
}

// SetAveraging sets the exponential averaging factor (0-1)
func (cm *CorrelationMeter) SetAveraging(factor float64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if factor >= 0 && factor <= 1 {
		cm.averaging = factor
	}
}

// SetPeakHoldTime sets the peak hold time in seconds
func (cm *CorrelationMeter) SetPeakHoldTime(seconds float64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.peakHoldTime = seconds
}

// Process updates the correlation meter with stereo samples
func (cm *CorrelationMeter) Process(samplesL, samplesR []float64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if len(samplesL) != len(samplesR) {
		return
	}
	
	for i := 0; i < len(samplesL); i++ {
		// Add samples to circular buffer
		cm.bufferL[cm.writePos] = samplesL[i]
		cm.bufferR[cm.writePos] = samplesR[i]
		
		cm.writePos = (cm.writePos + 1) % cm.windowSize
		if cm.count < cm.windowSize {
			cm.count++
		}
		
	}
	
	// Calculate correlation after processing all samples if buffer is full
	if cm.count == cm.windowSize {
		corr := cm.calculateCorrelation()
		
		// Apply exponential averaging
		cm.correlation = cm.correlation*cm.averaging + corr*(1-cm.averaging)
		
		// Update peak hold (for most negative correlation)
		if corr < cm.peakHold {
			cm.peakHold = corr
			cm.peakHoldCount = int(cm.peakHoldTime * cm.sampleRate / float64(cm.windowSize))
		} else {
			cm.peakHoldCount--
			if cm.peakHoldCount <= 0 {
				cm.peakHold = cm.correlation
				cm.peakHoldCount = 0
			}
		}
	}
}

// calculateCorrelation computes the Pearson correlation coefficient
func (cm *CorrelationMeter) calculateCorrelation() float64 {
	// Calculate means
	meanL := 0.0
	meanR := 0.0
	for i := 0; i < cm.count; i++ {
		meanL += cm.bufferL[i]
		meanR += cm.bufferR[i]
	}
	meanL /= float64(cm.count)
	meanR /= float64(cm.count)
	
	// Calculate correlation
	numerator := 0.0
	varL := 0.0
	varR := 0.0
	
	for i := 0; i < cm.count; i++ {
		diffL := cm.bufferL[i] - meanL
		diffR := cm.bufferR[i] - meanR
		
		numerator += diffL * diffR
		varL += diffL * diffL
		varR += diffR * diffR
	}
	
	// Avoid division by zero
	if varL == 0 || varR == 0 {
		if varL == 0 && varR == 0 {
			return 1.0 // Both channels silent, consider correlated
		}
		return 0.0 // One channel silent, no correlation
	}
	
	correlation := numerator / (math.Sqrt(varL) * math.Sqrt(varR))
	
	// Clamp to [-1, 1] to handle numerical errors
	if correlation > 1.0 {
		correlation = 1.0
	} else if correlation < -1.0 {
		correlation = -1.0
	}
	
	return correlation
}

// GetCorrelation returns the current correlation value (-1 to 1)
func (cm *CorrelationMeter) GetCorrelation() float64 {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.correlation
}

// GetPeakHold returns the peak hold correlation (most negative)
func (cm *CorrelationMeter) GetPeakHold() float64 {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.peakHold
}

// GetPhaseStatus returns a qualitative phase status
func (cm *CorrelationMeter) GetPhaseStatus() PhaseStatus {
	corr := cm.GetCorrelation()
	
	if corr > 0.9 {
		return PhaseInPhase
	} else if corr > 0.5 {
		return PhaseMostlyInPhase
	} else if corr > -0.5 {
		return PhasePartiallyCorrelated
	} else if corr > -0.9 {
		return PhaseMostlyOutOfPhase
	} else {
		return PhaseOutOfPhase
	}
}

// PhaseStatus represents the qualitative phase relationship
type PhaseStatus int

const (
	PhaseInPhase PhaseStatus = iota
	PhaseMostlyInPhase
	PhasePartiallyCorrelated
	PhaseMostlyOutOfPhase
	PhaseOutOfPhase
)

// String returns a string representation of the phase status
func (ps PhaseStatus) String() string {
	switch ps {
	case PhaseInPhase:
		return "In Phase"
	case PhaseMostlyInPhase:
		return "Mostly In Phase"
	case PhasePartiallyCorrelated:
		return "Partially Correlated"
	case PhaseMostlyOutOfPhase:
		return "Mostly Out of Phase"
	case PhaseOutOfPhase:
		return "Out of Phase"
	default:
		return "Unknown"
	}
}

// GetMonoCompatibility returns a mono compatibility score (0-1)
func (cm *CorrelationMeter) GetMonoCompatibility() float64 {
	corr := cm.GetCorrelation()
	// Map correlation from [-1, 1] to [0, 1]
	// -1 (out of phase) = 0 (bad mono compatibility)
	// +1 (in phase) = 1 (perfect mono compatibility)
	return (corr + 1.0) / 2.0
}

// Reset clears the correlation meter
func (cm *CorrelationMeter) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	// Clear buffers
	for i := range cm.bufferL {
		cm.bufferL[i] = 0
		cm.bufferR[i] = 0
	}
	
	// Reset state
	cm.writePos = 0
	cm.count = 0
	cm.correlation = 0
	cm.peakHold = 0
	cm.peakHoldCount = 0
}

// StereoFieldAnalyzer provides detailed stereo field analysis
type StereoFieldAnalyzer struct {
	correlation    *CorrelationMeter
	balanceMeter   *BalanceMeter
	widthMeter     *StereoWidthMeter
	sampleRate     float64
}

// NewStereoFieldAnalyzer creates a comprehensive stereo analyzer
func NewStereoFieldAnalyzer(windowSize int, sampleRate float64) *StereoFieldAnalyzer {
	return &StereoFieldAnalyzer{
		correlation:  NewCorrelationMeter(windowSize, sampleRate),
		balanceMeter: NewBalanceMeter(windowSize),
		widthMeter:   NewStereoWidthMeter(windowSize),
		sampleRate:   sampleRate,
	}
}

// Process analyzes stereo samples
func (sfa *StereoFieldAnalyzer) Process(samplesL, samplesR []float64) {
	sfa.correlation.Process(samplesL, samplesR)
	sfa.balanceMeter.Process(samplesL, samplesR)
	sfa.widthMeter.Process(samplesL, samplesR)
}

// GetAnalysis returns comprehensive stereo field analysis
func (sfa *StereoFieldAnalyzer) GetAnalysis() StereoFieldAnalysis {
	return StereoFieldAnalysis{
		Correlation:      sfa.correlation.GetCorrelation(),
		PhaseStatus:      sfa.correlation.GetPhaseStatus(),
		MonoCompatibility: sfa.correlation.GetMonoCompatibility(),
		Balance:          sfa.balanceMeter.GetBalance(),
		Width:            sfa.widthMeter.GetWidth(),
		WidthDB:          sfa.widthMeter.GetWidthDB(),
	}
}

// StereoFieldAnalysis contains stereo field measurements
type StereoFieldAnalysis struct {
	Correlation       float64
	PhaseStatus       PhaseStatus
	MonoCompatibility float64
	Balance           float64
	Width             float64
	WidthDB           float64
}

// BalanceMeter measures stereo balance
type BalanceMeter struct {
	windowSize int
	powerL     float64
	powerR     float64
	balance    float64
	averaging  float64
	mu         sync.Mutex
}

// NewBalanceMeter creates a new balance meter
func NewBalanceMeter(windowSize int) *BalanceMeter {
	return &BalanceMeter{
		windowSize: windowSize,
		averaging:  0.95,
	}
}

// Process updates the balance meter
func (bm *BalanceMeter) Process(samplesL, samplesR []float64) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	// Calculate power for each channel
	sumL := 0.0
	sumR := 0.0
	
	for i := 0; i < len(samplesL) && i < len(samplesR); i++ {
		sumL += samplesL[i] * samplesL[i]
		sumR += samplesR[i] * samplesR[i]
	}
	
	count := float64(len(samplesL))
	if count > 0 {
		powerL := sumL / count
		powerR := sumR / count
		
		// Apply averaging
		bm.powerL = bm.powerL*bm.averaging + powerL*(1-bm.averaging)
		bm.powerR = bm.powerR*bm.averaging + powerR*(1-bm.averaging)
		
		// Calculate balance (-1 = full left, 0 = center, 1 = full right)
		totalPower := bm.powerL + bm.powerR
		if totalPower > 0 {
			bm.balance = (bm.powerR - bm.powerL) / totalPower
		} else {
			bm.balance = 0
		}
	}
}

// GetBalance returns the stereo balance (-1 to 1)
func (bm *BalanceMeter) GetBalance() float64 {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.balance
}

// GetBalanceDB returns the balance in dB difference
func (bm *BalanceMeter) GetBalanceDB() float64 {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	if bm.powerL > 0 && bm.powerR > 0 {
		return 10.0 * math.Log10(bm.powerR/bm.powerL)
	}
	return 0
}

// StereoWidthMeter measures stereo width
type StereoWidthMeter struct {
	windowSize int
	midPower   float64
	sidePower  float64
	width      float64
	averaging  float64
	mu         sync.Mutex
}

// NewStereoWidthMeter creates a new stereo width meter
func NewStereoWidthMeter(windowSize int) *StereoWidthMeter {
	return &StereoWidthMeter{
		windowSize: windowSize,
		averaging:  0.95,
	}
}

// Process updates the width meter
func (swm *StereoWidthMeter) Process(samplesL, samplesR []float64) {
	swm.mu.Lock()
	defer swm.mu.Unlock()
	
	// Calculate mid/side signals
	sumMid := 0.0
	sumSide := 0.0
	
	for i := 0; i < len(samplesL) && i < len(samplesR); i++ {
		mid := (samplesL[i] + samplesR[i]) * 0.5
		side := (samplesL[i] - samplesR[i]) * 0.5
		
		sumMid += mid * mid
		sumSide += side * side
	}
	
	count := float64(len(samplesL))
	if count > 0 {
		midPower := sumMid / count
		sidePower := sumSide / count
		
		// Apply averaging
		swm.midPower = swm.midPower*swm.averaging + midPower*(1-swm.averaging)
		swm.sidePower = swm.sidePower*swm.averaging + sidePower*(1-swm.averaging)
		
		// Calculate width (0 = mono, 1 = normal stereo, >1 = extra wide)
		totalPower := swm.midPower + swm.sidePower
		if totalPower > 0 {
			swm.width = math.Sqrt(swm.sidePower / swm.midPower)
		} else {
			swm.width = 0
		}
	}
}

// GetWidth returns the stereo width (0 = mono, 1 = normal, >1 = wide)
func (swm *StereoWidthMeter) GetWidth() float64 {
	swm.mu.Lock()
	defer swm.mu.Unlock()
	return swm.width
}

// GetWidthDB returns the width as side/mid ratio in dB
func (swm *StereoWidthMeter) GetWidthDB() float64 {
	swm.mu.Lock()
	defer swm.mu.Unlock()
	
	if swm.midPower > 0 && swm.sidePower > 0 {
		return 10.0 * math.Log10(swm.sidePower/swm.midPower)
	}
	return -math.Inf(1)
}