package profile

import "testing"

func TestMaskPhone(t *testing.T) {
	for in, want := range map[string]string{
		"0812-3456-2210":    "0812-xxxx-2210",
		"+62 812 3456 2210": "6281-xxxx-2210",
		"081234562210":      "0812-xxxx-2210",
		"12345":             "xxx45",
		"":                  "",
	} {
		if got := MaskPhone(in); got != want {
			t.Errorf("MaskPhone(%q) = %q, seharusnya %q", in, got, want)
		}
	}
}
