package trialquiz

import "testing"

func TestNormalizeWhatsApp(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"081234567890", "6281234567890", true},
		{"0812-3456-7890", "6281234567890", true},
		{"+62 812 3456 7890", "6281234567890", true},
		{"6281234567890", "6281234567890", true},
		{"81234567890", "6281234567890", true},
		{"(0812) 3456.7890", "6281234567890", true},
		{"0211234567", "", false},        // telepon rumah (bukan 08)
		{"0812", "", false},              // terlalu pendek
		{"08123456789012345", "", false}, // terlalu panjang
		{"0812abc4567", "", false},       // ada huruf
		{"62+812345678", "", false},      // + di tengah
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := NormalizeWhatsApp(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("NormalizeWhatsApp(%q) = %q, %v; want %q, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
