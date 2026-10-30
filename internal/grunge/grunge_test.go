package grunge

import "testing"

func TestMaps(t *testing.T) {
	names := Names()
	if len(names) == 0 {
		t.Fatal("no maps embedded")
	}
	if _, err := PNG(names[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := PNG("../grunge"); err == nil {
		t.Fatal("path escaped the maps folder")
	}
}
