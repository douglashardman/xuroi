package auth

import "testing"

func TestReservedDisplayName(t *testing.T) {
	set := BuildReservedSet([]string{"puttertalk"})
	cases := []struct {
		name    string
		want    bool
	}{
		{"Admin", true},
		{"ADMIN", true},
		{" moderator ", true},
		{"PutterTalk", true},
		{"Doug", false},
		{"Gear Tester", false},
	}
	for _, tc := range cases {
		if got := reservedDisplayName(tc.name, set); got != tc.want {
			t.Fatalf("%q reserved=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestBuildReservedSetMergesSiteNames(t *testing.T) {
	set := BuildReservedSet([]string{"MrDoug"})
	if _, ok := set["mrdoug"]; !ok {
		t.Fatal("expected site reserved name")
	}
	if _, ok := set["admin"]; !ok {
		t.Fatal("expected default reserved name")
	}
}