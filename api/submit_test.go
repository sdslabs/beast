package api

import "testing"

func TestScoreAfterPointChangeDoesNotUnderflow(t *testing.T) {
	tests := []struct {
		name              string
		current, new, old uint
		want              uint
	}{
		{name: "increase", current: 100, new: 75, old: 50, want: 125},
		{name: "decrease", current: 100, new: 25, old: 50, want: 75},
		{name: "clamp at zero", current: 10, new: 0, old: 50, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := scoreAfterPointChange(test.current, test.new, test.old); got != test.want {
				t.Fatalf("scoreAfterPointChange() = %d, want %d", got, test.want)
			}
		})
	}
}
