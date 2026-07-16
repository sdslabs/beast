package remoteManager

import "testing"

func TestShellJoinQuotesEveryArgument(t *testing.T) {
	got := shellJoin([]string{"docker", "name; touch /tmp/pwned", "it's", ""})
	want := `'docker' 'name; touch /tmp/pwned' 'it'"'"'s' ''`
	if got != want {
		t.Fatalf("shellJoin() = %q, want %q", got, want)
	}
}

func TestCappedCommandBufferDiscardsExcess(t *testing.T) {
	buffer := &cappedCommandBuffer{remaining: 3}
	written, err := buffer.Write([]byte("abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	if written != 6 || buffer.buffer.String() != "abc" || !buffer.truncated {
		t.Fatalf("unexpected buffer state: written=%d value=%q truncated=%t", written, buffer.buffer.String(), buffer.truncated)
	}
}
