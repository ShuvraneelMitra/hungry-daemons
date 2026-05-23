package profiler

import (
	"log"
	"runtime"
	"sync"
	"time"
)

type Metadata struct {
	Timestamp time.Time
	truncated bool
	numGoroutines int
	stackDump []byte
}

type Sampler interface {
	/*
	This sets off a goroutine which periodically samples the whole
	program with the given samplingFrequency. To use this the caller
	must first set the sampling frequency in Hertz.
	*/
	Sample(stop <-chan any) <-chan Metadata // not a pointer since we don't modify this

	SetSamplingFrequency(int)
	SetMaxBufferSize(int)
}

type GoRoutineSampler struct {
	// Collects a snapshot of all goroutine stacks every (1/sampling_f) seconds.
	sampling_f int

	MAX_BUFFER_SIZE int
	MAX_CHANNEL_SIZE int
	/*
	The above variables might be susceptible to race conditions but we will leave
	them as is since the intention is to only update these variables BEFORE the 
	start of any profiling session.
	*/

	TotalTruncations int // For collecting statistics
	TotalSamples int
	TotalDrops int

	mtx sync.RWMutex
}

func NewGoRoutineSampler() *GoRoutineSampler {
	return &GoRoutineSampler{
		sampling_f: 0,
		MAX_BUFFER_SIZE: 1 << 30, // DEFAULT: 1 GB
		MAX_CHANNEL_SIZE: 100,
	}
}

func (sampler *GoRoutineSampler) Sample(stop <-chan any) <-chan Metadata {
	if sampler.sampling_f == 0 {
		log.Fatal("Set the sampling frequency on the goroutine sampler!")
	}

	dataStream := make(chan Metadata, sampler.MAX_CHANNEL_SIZE)

	go func() {
		defer close(dataStream)
		ticker := time.NewTicker(time.Second / time.Duration(sampler.sampling_f))
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return 
			case <-ticker.C: // ticker.C is an unbuffered channel
				metadata := Metadata{
					Timestamp: time.Now(),
					truncated: false,
					stackDump: make([]byte, 5 * (1 << 20)), // start with 5 MB, truncate later if required
					numGoroutines: runtime.NumGoroutine(),
				}
				for {
					n := runtime.Stack(metadata.stackDump, true)
					if len(metadata.stackDump) == sampler.MAX_BUFFER_SIZE && n == len(metadata.stackDump) {
						metadata.truncated = true

						sampler.mtx.Lock()
						sampler.TotalTruncations += 1
						sampler.mtx.Unlock()
						break
					} else if n < len(metadata.stackDump) {
						metadata.stackDump = metadata.stackDump[ : n]
						break
					} else {
						bufsize := min(2 * len(metadata.stackDump), sampler.MAX_BUFFER_SIZE)
						metadata.stackDump = make([]byte, bufsize)
					}
				}
				
				timeout := time.NewTimer(1 * time.Second)
				defer timeout.Stop()

				select {
				case dataStream<- metadata:
					if !timeout.Stop() {
						<-timeout.C
					}
				case <-timeout.C: // will wait till just after the new sampling point
					sampler.mtx.Lock()
					// if there are too many queued stacktraces, we start dropping consequent traces
					sampler.TotalDrops++
					sampler.mtx.Unlock()
					continue
				}

				sampler.mtx.Lock()
				sampler.TotalSamples += 1
				sampler.mtx.Unlock()
			}
		}
	}()
	return dataStream
}

func (sampler *GoRoutineSampler) SetSamplingFrequency(f int) {
	sampler.sampling_f = f
	sampler.MAX_CHANNEL_SIZE = 1 * sampler.sampling_f
	// Tolerate a maximum delay of 1 second in the processing of a stack trace
}

func (sampler *GoRoutineSampler) SetMaxBufferSize(sz int) {
	sampler.MAX_BUFFER_SIZE = sz
}



