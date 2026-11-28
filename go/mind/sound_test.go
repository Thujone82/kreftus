package main

import (
	"encoding/binary"
	"testing"
)

func TestSynthWAV(t *testing.T) {
	gapSamples := (wavSampleRate * wavGapMs) / 1000

	for _, id := range allSoundIDs {
		spec, ok := soundSpecs[id]
		if !ok {
			t.Fatalf("missing soundSpec for soundID %d", id)
		}
		wav := synthWAV(spec.notes, spec.square)
		if len(wav) < 44 {
			t.Fatalf("soundID %d WAV too short: %d", id, len(wav))
		}
		if string(wav[0:4]) != "RIFF" {
			t.Errorf("soundID %d missing RIFF header: %q", id, wav[0:4])
		}
		if string(wav[8:12]) != "WAVE" {
			t.Errorf("soundID %d missing WAVE header: %q", id, wav[8:12])
		}
		if string(wav[12:16]) != "fmt " {
			t.Errorf("soundID %d missing fmt header: %q", id, wav[12:16])
		}
		if string(wav[36:40]) != "data" {
			t.Errorf("soundID %d missing data header: %q", id, wav[36:40])
		}

		riffSize := binary.LittleEndian.Uint32(wav[4:8])
		if int(riffSize) != len(wav)-8 {
			t.Errorf("soundID %d RIFF size = %d, want %d", id, riffSize, len(wav)-8)
		}

		expectedSamples := 0
		for i, n := range spec.notes {
			if n.ms > 0 {
				expectedSamples += (wavSampleRate * n.ms) / 1000
			}
			if i+1 < len(spec.notes) {
				expectedSamples += gapSamples
			}
		}
		dataSize := binary.LittleEndian.Uint32(wav[40:44])
		if int(dataSize) != expectedSamples*2 {
			t.Errorf("soundID %d data size = %d, want %d", id, dataSize, expectedSamples*2)
		}
		if len(wav) != 44+int(dataSize) {
			t.Errorf("soundID %d len(wav) = %d, want %d", id, len(wav), 44+int(dataSize))
		}
	}
}
