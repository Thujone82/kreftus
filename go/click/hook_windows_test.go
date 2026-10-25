//go:build windows

package main

import (
	"testing"
	"time"
)

func TestNotePointerSkipsClicksOnWindow(t *testing.T) {
	recOn = false
	rawN = 0
	haveCursor = false
	winL, winT, winR, winB = 0, 0, 10, 10
	recStart = time.Now()
	recOn = true
	t.Cleanup(func() {
		recOn = false
		rawN = 0
		haveCursor = false
	})

	notePointer(15, 25, 0, 0)      // travel is not stored
	notePointer(15, 25, 0x0001, 0) // left down, outside the window
	notePointer(3, 3, 0x0001, 0)   // left down on the window
	notePointer(40, 50, 0x0400, -120)

	if rawN != 2 {
		t.Fatalf("count %d", rawN)
	}
	if rawSlots[0].Kind != LeftDown || rawSlots[0].X != 15 {
		t.Fatalf("down %+v", rawSlots[0])
	}
	if rawSlots[1].Kind != Wheel || rawSlots[1].Wheel != -120 || rawSlots[1].X != 40 {
		t.Fatalf("wheel %+v", rawSlots[1])
	}
	for i := 0; i < rawN; i++ {
		if rawSlots[i].Kind == LeftDown && rawSlots[i].X == 3 {
			t.Fatalf("click on window was recorded: %+v", rawSlots[i])
		}
	}
}

func TestApplyKeyTapAndHold(t *testing.T) {
	recOn = false
	rawN = 0
	haveCursor = false
	winL, winT, winR, winB = 0, 0, 0, 0
	recStart = time.Now()
	recOn = true
	t.Cleanup(func() {
		recOn = false
		rawN = 0
	})

	tick := uint32(0)
	down := applyKey(false, keySample{down: true}, LeftDown, LeftUp, 8, 9, tick)
	if !down || rawN != 1 || rawSlots[0].Kind != LeftDown {
		t.Fatalf("down state %v count %d %+v", down, rawN, rawSlots[0])
	}
	down = applyKey(true, keySample{down: true}, LeftDown, LeftUp, 8, 9, tick)
	if !down || rawN != 1 {
		t.Fatalf("hold added an event: %d", rawN)
	}
	down = applyKey(false, keySample{pressed: true}, LeftDown, LeftUp, 12, 13, 5)
	if down || rawN != 3 || rawSlots[1].Kind != LeftDown || rawSlots[2].Kind != LeftUp {
		t.Fatalf("tap down=%v count=%d %+v %+v", down, rawN, rawSlots[1], rawSlots[2])
	}
}
