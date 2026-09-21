package ui

import "testing"

// The regression behind v0.47.0's link-completion fixes: "popup vid en
// länk i edit mode när jag rör markören över länken" — arrow keys
// skimming across an existing [[link]] must never summon the suggestion
// popup; only a key that writes must.

func TestMovingOverALinkNeverOpensCompletion(t *testing.T) {
	m := newTestModel(t)
	onWelcome(m)
	press(m, "enter") // open the editor on Welcome
	if m.editor == nil {
		t.Fatal("no editor")
	}
	// The note's one line is "# Welcome" followed by "See [[Stoic]] and
	// [[Missing]] #start". Skim right across the whole line and past
	// both links: no popup.
	for i := 0; i < 60; i++ {
		press(m, "right")
		if m.complete != nil {
			t.Fatalf("arrow over a link opened the popup at step %d", i)
		}
	}
	for i := 0; i < 60; i++ {
		press(m, "down")
		if m.complete != nil {
			t.Fatalf("down over a wrapped link opened the popup at step %d", i)
		}
	}
	// And the popup still opens for real typing inside brackets.
	typeText(m, "\n[[Sto")
	if m.complete == nil {
		t.Fatal("typing inside [[ didn't open the popup")
	}
	// Esc closes it, and moving again never reopens it.
	press(m, "esc")
	if m.complete != nil {
		t.Fatal("esc didn't close the popup")
	}
	press(m, "left", "right", "left")
	if m.complete != nil {
		t.Fatal("moving after Esc reopened the popup")
	}
	// A dismissed popup (noComplete) stays down under movement too.
	typeText(m, "ic]]")
	press(m, "esc")
	for i := 0; i < 10; i++ {
		press(m, "right")
		if m.complete != nil {
			t.Fatal("movement resurrected a dismissed popup")
		}
	}
}

func TestTypingInsideBracketsKeepsCompletionAlive(t *testing.T) {
	m := newTestModel(t)
	onWelcome(m)
	press(m, "enter")
	if m.editor == nil {
		t.Fatal("no editor")
	}
	typeText(m, "\n[[Sto")
	if m.complete == nil {
		t.Fatal("typing inside [[ didn't open the popup")
	}
	// More typing refreshes it (the query grows).
	typeText(m, "i")
	if m.complete == nil {
		t.Fatal("the popup vanished mid-typing")
	}
	// Closing the brackets closes the popup — that's the editor closing
	// it on ]] — which is correct and must keep working.
	typeText(m, "c]]")
	if m.complete != nil {
		t.Fatal("the popup stayed open past ]]")
	}
}
