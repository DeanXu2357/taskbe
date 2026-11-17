package httpapi

import "testing"

func TestCursorRoundTrip(t *testing.T) {
	c := encodeCursor(2)
	if c != "eyJhZnRlcl9pZCI6Mn0" {
		t.Errorf("encodeCursor(2) = %q, want eyJhZnRlcl9pZCI6Mn0", c)
	}
	id, err := decodeCursor(c)
	if err != nil || id != 2 {
		t.Errorf("decodeCursor(%q) = %d, %v, want 2, nil", c, id, err)
	}
}
