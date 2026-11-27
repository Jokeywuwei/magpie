package usage

import (
	"testing"
	"testing/synctest"
)

// A timer armed inside a testing/synctest bubble ends with that bubble's
// world. A linger outside it must arm a timer of its own rather than reset
// the dead world's, which the Go runtime makes a fatal error. The two
// gateway-mode tests in internal/gui hit exactly this, one after the other.
func TestLingerAfterSynctestBubble(t *testing.T) {
	t.Cleanup(ReleaseScratch)
	ReleaseScratch()
	synctest.Test(t, func(t *testing.T) {
		lingerScratch() // arms a timer in the bubble, which ends here
	})
	lingerScratch() // the old code's Reset made this fatal
	scratchSweep.Lock()
	armed, seq := scratchSweep.armed, scratchSweep.seq
	scratchSweep.Unlock()
	if !armed {
		t.Fatal("no sweep armed after the bubble")
	}
	lingerScratch()
	scratchSweep.Lock()
	same := scratchSweep.seq == seq
	scratchSweep.Unlock()
	if !same {
		t.Fatal("a second linger armed a timer of its own")
	}
	ReleaseScratch()
	scratchSweep.Lock()
	armed = scratchSweep.armed
	scratchSweep.Unlock()
	if armed {
		t.Fatal("ReleaseScratch left the sweep armed")
	}
}
