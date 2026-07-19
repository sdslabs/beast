package cr

import "testing"

func TestCappedBufferDiscardsExcessWithoutBlockingWriter(t *testing.T) {
	buffer := &cappedBuffer{remaining: 4}
	data := []byte("abcdefgh")
	written, err := buffer.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if written != len(data) {
		t.Fatalf("reported written bytes = %d, want %d", written, len(data))
	}
	if got := buffer.String(); got != "abcd" {
		t.Fatalf("buffer = %q", got)
	}
	if !buffer.Truncated() {
		t.Fatal("expected truncation marker")
	}
}
