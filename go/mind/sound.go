package main

import (
	"encoding/binary"
	"math"
	"os"
	"runtime"
)

type soundID int

const (
	sndClick soundID = iota
	sndDelete
	sndInvalid
	sndSubmit
	sndExact
	sndPartial
	sndMiss
	sndReveal
	sndWin
	sndLose
	sndTheme
	sndStart
)

var allSoundIDs = []soundID{
	sndClick, sndDelete, sndInvalid, sndSubmit,
	sndExact, sndPartial, sndMiss, sndReveal,
	sndWin, sndLose, sndTheme, sndStart,
}

type note struct {
	freq float64
	ms   int
}

type soundSpec struct {
	notes  []note
	square bool
}

var soundSpecs = map[soundID]soundSpec{
	sndClick:   {notes: []note{{1200, 22}}},
	sndDelete:  {notes: []note{{600, 30}}},
	sndInvalid: {notes: []note{{150, 140}}, square: true},
	sndSubmit:  {notes: []note{{660, 40}, {990, 50}}},
	sndExact:   {notes: []note{{1046, 70}}},
	sndPartial: {notes: []note{{784, 70}}},
	sndMiss:    {notes: []note{{220, 120}}},
	sndReveal:  {notes: []note{{880, 50}}},
	sndWin:     {notes: []note{{523, 110}, {659, 110}, {784, 110}, {1046, 320}}},
	sndLose:    {notes: []note{{392, 220}, {330, 220}, {262, 420}}},
	sndTheme:   {notes: []note{{880, 40}, {1320, 60}}},
	sndStart:   {notes: []note{{523, 70}, {784, 110}}},
}

const (
	wavSampleRate = 22050
	wavGapMs      = 10
	wavAttackMs   = 4
	wavAmplitude  = 0.35
)

// synthWAV synthesizes a 22050 Hz mono 16-bit PCM WAV buffer for the given note sequence.
func synthWAV(notes []note, square bool) []byte {
	gapSamples := (wavSampleRate * wavGapMs) / 1000
	attackSamples := (wavSampleRate * wavAttackMs) / 1000

	totalSamples := 0
	for i, n := range notes {
		if n.ms > 0 {
			totalSamples += (wavSampleRate * n.ms) / 1000
		}
		if i+1 < len(notes) {
			totalSamples += gapSamples
		}
	}

	dataSize := totalSamples * 2
	out := make([]byte, 44+dataSize)

	copy(out[0:4], "RIFF")
	binary.LittleEndian.PutUint32(out[4:8], uint32(36+dataSize))
	copy(out[8:12], "WAVE")
	copy(out[12:16], "fmt ")
	binary.LittleEndian.PutUint32(out[16:20], 16)
	binary.LittleEndian.PutUint16(out[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(out[22:24], 1) // Mono
	binary.LittleEndian.PutUint32(out[24:28], wavSampleRate)
	binary.LittleEndian.PutUint32(out[28:32], wavSampleRate*2)
	binary.LittleEndian.PutUint16(out[32:34], 2)  // BlockAlign
	binary.LittleEndian.PutUint16(out[34:36], 16) // BitsPerSample
	copy(out[36:40], "data")
	binary.LittleEndian.PutUint32(out[40:44], uint32(dataSize))

	offset := 44
	for ni, n := range notes {
		nSamples := 0
		if n.ms > 0 {
			nSamples = (wavSampleRate * n.ms) / 1000
		}
		for i := 0; i < nSamples; i++ {
			var sample float64
			if n.freq > 0 {
				t := float64(i) / float64(wavSampleRate)
				phase := 2 * math.Pi * n.freq * t
				if square {
					if math.Sin(phase) >= 0 {
						sample = 1
					} else {
						sample = -1
					}
				} else {
					sample = math.Sin(phase)
				}
				env := math.Pow(0.10, float64(i)/float64(nSamples))
				if attackSamples > 0 && i < attackSamples {
					env *= float64(i+1) / float64(attackSamples)
				}
				sample *= wavAmplitude * env
			}
			pcm := int16(math.Round(sample * 32767))
			binary.LittleEndian.PutUint16(out[offset:offset+2], uint16(pcm))
			offset += 2
		}
		if ni+1 < len(notes) {
			offset += gapSamples * 2
		}
	}
	return out
}

type player struct {
	muted bool
}

func (p *player) play(id soundID) {
	if p == nil || p.muted {
		return
	}
	if runtime.GOOS == "windows" && !remoteSSH() {
		playPCM(id)
		return
	}
	switch id {
	case sndWin, sndLose, sndInvalid:
		_, _ = os.Stdout.WriteString("\a")
	}
}
