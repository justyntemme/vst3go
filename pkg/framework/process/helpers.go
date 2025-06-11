package process

// ProcessChannels processes all available channels with the given function
func (ctx *Context) ProcessChannels(fn func(ch int, input, output []float32)) {
	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}
	
	for ch := 0; ch < numChannels; ch++ {
		fn(ch, ctx.Input[ch], ctx.Output[ch])
	}
}

// ProcessStereo processes up to 2 channels (stereo) with the given function
func (ctx *Context) ProcessStereo(fn func(ch int, input, output []float32)) {
	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}
	if numChannels > 2 {
		numChannels = 2 // Limit to stereo
	}
	
	for ch := 0; ch < numChannels; ch++ {
		fn(ch, ctx.Input[ch], ctx.Output[ch])
	}
}

// ProcessMono processes only the first channel
func (ctx *Context) ProcessMono(fn func(input, output []float32)) {
	if ctx.NumInputChannels() > 0 && ctx.NumOutputChannels() > 0 {
		fn(ctx.Input[0], ctx.Output[0])
	}
}

// ProcessSamples processes each sample across all channels
func (ctx *Context) ProcessSamples(fn func(sample int, inputs, outputs []float32)) {
	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}
	
	numSamples := ctx.NumSamples()
	
	// Temporary slices to avoid allocations
	inputs := make([]float32, numChannels)
	outputs := make([]float32, numChannels)
	
	for s := 0; s < numSamples; s++ {
		// Gather inputs
		for ch := 0; ch < numChannels; ch++ {
			inputs[ch] = ctx.Input[ch][s]
		}
		
		// Process
		fn(s, inputs, outputs)
		
		// Write outputs
		for ch := 0; ch < numChannels; ch++ {
			ctx.Output[ch][s] = outputs[ch]
		}
	}
}

// ProcessChannelsSeparately processes each channel independently with its own function
func (ctx *Context) ProcessChannelsSeparately(fns ...func(input, output []float32)) {
	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}
	if len(fns) < numChannels {
		numChannels = len(fns)
	}
	
	for ch := 0; ch < numChannels; ch++ {
		fns[ch](ctx.Input[ch], ctx.Output[ch])
	}
}

// CopyInputToOutput copies input to output for all channels
func (ctx *Context) CopyInputToOutput() {
	ctx.ProcessChannels(func(ch int, input, output []float32) {
		copy(output, input)
	})
}

// GetNumChannels returns the minimum of input and output channels
func (ctx *Context) GetNumChannels() int {
	numChannels := ctx.NumInputChannels()
	if ctx.NumOutputChannels() < numChannels {
		numChannels = ctx.NumOutputChannels()
	}
	return numChannels
}

// GetNumStereoChannels returns the number of channels capped at 2
func (ctx *Context) GetNumStereoChannels() int {
	numChannels := ctx.GetNumChannels()
	if numChannels > 2 {
		return 2
	}
	return numChannels
}