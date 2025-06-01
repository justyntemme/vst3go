package plugin

import (
	"testing"
	"time"

	"github.com/justyntemme/vst3go/pkg/framework/bus"
	"github.com/justyntemme/vst3go/pkg/framework/param"
	"github.com/justyntemme/vst3go/pkg/framework/process"
)

// BenchmarkBufferedVsUnbuffered compares performance of buffered vs unbuffered processing
func BenchmarkUnbufferedProcessor(b *testing.B) {
	proc := &mockProcessor{
		params: param.NewRegistry(),
		buses:  bus.NewStereoConfiguration(),
	}
	
	proc.Initialize(44100, 512)
	proc.SetActive(true)
	
	ctx := &process.Context{
		Input:      make([][]float32, 2),
		Output:     make([][]float32, 2),
		SampleRate: 44100,
	}
	for i := 0; i < 2; i++ {
		ctx.Input[i] = make([]float32, 512)
		ctx.Output[i] = make([]float32, 512)
	}
	// Number of samples is determined by buffer size
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		proc.ProcessAudio(ctx)
	}
}

func BenchmarkBufferedProcessor(b *testing.B) {
	proc := &mockProcessor{
		params: param.NewRegistry(),
		buses:  bus.NewStereoConfiguration(),
	}
	
	buffered := NewBufferedProcessor(proc, 2)
	buffered.Initialize(44100, 512)
	buffered.SetActive(true)
	
	ctx := &process.Context{
		Input:      make([][]float32, 2),
		Output:     make([][]float32, 2),
		SampleRate: 44100,
	}
	for i := 0; i < 2; i++ {
		ctx.Input[i] = make([]float32, 512)
		ctx.Output[i] = make([]float32, 512)
	}
	// Number of samples is determined by buffer size
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		buffered.ProcessAudio(ctx)
	}
}

// BenchmarkBufferOverhead measures the overhead of buffering alone
func BenchmarkBufferOverhead(b *testing.B) {
	proc := &mockProcessor{
		params: param.NewRegistry(),
		buses:  bus.NewStereoConfiguration(),
	}
	
	buffered := NewBufferedProcessor(proc, 2)
	buffered.Initialize(44100, 512)
	buffered.SetActive(true)
	
	ctx := &process.Context{
		Input:      make([][]float32, 2),
		Output:     make([][]float32, 2),
		SampleRate: 44100,
	}
	for i := 0; i < 2; i++ {
		ctx.Input[i] = make([]float32, 512)
		ctx.Output[i] = make([]float32, 512)
	}
	// Number of samples is determined by buffer size
	
	// Pre-fill the buffer
	for i := 0; i < 100; i++ {
		buffered.ProcessAudio(ctx)
	}
	
	b.ResetTimer()
	
	start := time.Now()
	for i := 0; i < b.N; i++ {
		buffered.ProcessAudio(ctx)
	}
	elapsed := time.Since(start)
	
	b.ReportMetric(float64(b.N)*512/elapsed.Seconds(), "samples/sec")
}

// BenchmarkGCPressure measures allocation patterns
func BenchmarkGCPressure(b *testing.B) {
	proc := &mockProcessor{
		params: param.NewRegistry(),
		buses:  bus.NewStereoConfiguration(),
	}
	
	buffered := NewBufferedProcessor(proc, 2)
	buffered.Initialize(44100, 512)
	buffered.SetActive(true)
	
	ctx := &process.Context{
		Input:      make([][]float32, 2),
		Output:     make([][]float32, 2),
		SampleRate: 44100,
	}
	for i := 0; i < 2; i++ {
		ctx.Input[i] = make([]float32, 512)
		ctx.Output[i] = make([]float32, 512)
	}
	// Number of samples is determined by buffer size
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		buffered.ProcessAudio(ctx)
	}
}

// BenchmarkConcurrentProcessing measures performance under concurrent load
func BenchmarkConcurrentProcessing(b *testing.B) {
	numProcessors := 4
	processors := make([]*BufferedProcessor, numProcessors)
	contexts := make([]*process.Context, numProcessors)
	
	for i := 0; i < numProcessors; i++ {
		proc := &mockProcessor{
			params: param.NewRegistry(),
			buses:  bus.NewStereoConfiguration(),
		}
		
		processors[i] = NewBufferedProcessor(proc, 2)
		processors[i].Initialize(44100, 512)
		processors[i].SetActive(true)
		
		ctx := &process.Context{
			Input:      make([][]float32, 2),
			Output:     make([][]float32, 2),
			SampleRate: 44100,
		}
		for j := 0; j < 2; j++ {
			ctx.Input[j] = make([]float32, 512)
			ctx.Output[j] = make([]float32, 512)
		}
		// Number of samples is determined by buffer size
		contexts[i] = ctx
	}
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			processors[i%numProcessors].ProcessAudio(contexts[i%numProcessors])
			i++
		}
	})
}