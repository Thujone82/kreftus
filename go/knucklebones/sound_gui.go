//go:build windows || darwin || (linux && cgo)

package main

import (
	"encoding/binary"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const soundSampleRate = 22050

var (
	soundOnce sync.Once
	soundCtx  *audio.Context
	soundPCM  map[soundID][]byte
)

func initSounds() {
	soundPCM = map[soundID][]byte{
		sndClick:   synthToneSeq([]toneSpec{{freq0: 760, freq1: 520, durMs: 28, amp: 0.14, wave: waveTriangle}}),
		sndPlace:   synthToneSeq([]toneSpec{{freq0: 250, freq1: 115, durMs: 75, amp: 0.28, wave: waveTriangle}, {freq0: 540, freq1: 220, durMs: 24, amp: 0.10, wave: waveNoise}}),
		sndDestroy: synthToneSeq([]toneSpec{{freq0: 360, freq1: 85, durMs: 130, amp: 0.30, wave: waveNoise}, {freq0: 220, freq1: 110, durMs: 110, amp: 0.24, wave: waveSquare}}),
		sndTurn:    synthToneSeq([]toneSpec{{freq0: 523.25, freq1: 523.25, durMs: 95, amp: 0.16, wave: waveSine}}),
		sndWin: synthToneSeq([]toneSpec{
			{startMs: 0, freq0: 523.25, freq1: 523.25, durMs: 120, amp: 0.22, wave: waveSine},
			{startMs: 85, freq0: 659.25, freq1: 659.25, durMs: 130, amp: 0.22, wave: waveSine},
			{startMs: 170, freq0: 783.99, freq1: 783.99, durMs: 210, amp: 0.24, wave: waveSine},
		}),
		sndLose: synthToneSeq([]toneSpec{
			{startMs: 0, freq0: 329.63, freq1: 311.13, durMs: 140, amp: 0.20, wave: waveTriangle},
			{startMs: 120, freq0: 246.94, freq1: 220.00, durMs: 230, amp: 0.22, wave: waveTriangle},
		}),
	}
	if ctx := audio.CurrentContext(); ctx != nil {
		soundCtx = ctx
	} else {
		soundCtx = audio.NewContext(soundSampleRate)
	}
}

func playPCMSound(id soundID) bool {
	if id == sndNone {
		return true
	}
	if headlessAudio() {
		return false
	}
	soundOnce.Do(initSounds)
	pcm := soundPCM[id]
	if soundCtx == nil || len(pcm) == 0 {
		return false
	}
	p := soundCtx.NewPlayerFromBytes(pcm)
	p.Play()
	return true
}

const (
	waveSine = iota
	waveTriangle
	waveSquare
	waveNoise
)

type toneSpec struct {
	startMs float64
	freq0   float64
	freq1   float64
	durMs   float64
	amp     float64
	wave    int
}

func synthToneSeq(specs []toneSpec) []byte {
	var totalMs float64
	for _, s := range specs {
		if end := s.startMs + s.durMs; end > totalMs {
			totalMs = end
		}
	}
	n := int(math.Ceil(totalMs * float64(soundSampleRate) / 1000.0))
	if n <= 0 {
		return nil
	}
	mix := make([]float64, n)
	var noise uint32 = 0x9e3779b9
	for _, s := range specs {
		startIdx := int(s.startMs * float64(soundSampleRate) / 1000.0)
		count := int(s.durMs * float64(soundSampleRate) / 1000.0)
		if count <= 0 {
			continue
		}
		phase := 0.0
		attack := math.Min(float64(count)*0.12, 0.006*float64(soundSampleRate))
		for i := 0; i < count && startIdx+i < n; i++ {
			t := float64(i) / float64(count)
			freq := s.freq0 + (s.freq1-s.freq0)*t
			phase += 2 * math.Pi * freq / float64(soundSampleRate)

			env := 1.0 - t
			env *= env
			if float64(i) < attack && attack > 0 {
				env *= float64(i) / attack
			}

			var sample float64
			switch s.wave {
			case waveSine:
				sample = math.Sin(phase)
			case waveTriangle:
				p := math.Mod(phase/(2*math.Pi), 1.0)
				sample = 1.0 - 4.0*math.Abs(math.Round(p-0.25)-(p-0.25))
			case waveSquare:
				if math.Sin(phase) >= 0 {
					sample = 0.65
				} else {
					sample = -0.65
				}
			case waveNoise:
				noise ^= noise << 13
				noise ^= noise >> 17
				noise ^= noise << 5
				sample = float64(int32(noise)) / math.MaxInt32
			}
			mix[startIdx+i] += sample * s.amp * env
		}
	}

	out := make([]byte, n*4)
	for i, v := range mix {
		if v > 0.95 {
			v = 0.95
		} else if v < -0.95 {
			v = -0.95
		}
		iv := uint16(int16(v * 32760))
		off := i * 4
		binary.LittleEndian.PutUint16(out[off:], iv)
		binary.LittleEndian.PutUint16(out[off+2:], iv)
	}
	return out
}
