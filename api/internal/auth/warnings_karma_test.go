package auth

import "testing"

func TestWarningKarmaDeduction(t *testing.T) {
	tests := []struct {
		karma int
		want  int
	}{
		{0, 0},
		{1, 0},
		{4, 0},  // 20% = 0.8 → 1 → even 0
		{5, 0},  // 20% = 1 → even 0
		{10, 2},
		{15, 2}, // 20% = 3 → even 2
		{50, 10},
		{100, 20},
		{101, 20}, // 20.2 → 20
		{99, 20},  // 19.8 → 20
	}
	for _, tc := range tests {
		got := warningKarmaDeduction(tc.karma)
		if got != tc.want {
			t.Errorf("warningKarmaDeduction(%d) = %d, want %d", tc.karma, got, tc.want)
		}
		if got%2 != 0 && got > 0 {
			t.Errorf("warningKarmaDeduction(%d) = %d, not even", tc.karma, got)
		}
	}
}