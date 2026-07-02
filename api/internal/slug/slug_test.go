package slug

import "testing"

func TestFromDisplayName(t *testing.T) {
	cases := map[string]string{
		"gear_tester":   "gear-tester",
		"Green Reader":  "green-reader",
		"  Doug  ":       "doug",
	}
	for in, want := range cases {
		if got := FromDisplayName(in); got != want {
			t.Fatalf("%q => %q, want %q", in, got, want)
		}
	}
}