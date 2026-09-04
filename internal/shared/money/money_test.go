package money

import "testing"

func TestIDRString(t *testing.T) {
	cases := []struct {
		in   IDR
		want string
	}{
		{Zero, "Rp 0"},
		{ParseIDR(250_000), "Rp 250.000"},
		{ParseIDR(1_000_000), "Rp 1.000.000"},
		{ParseIDR(1_234_567_890), "Rp 1.234.567.890"},
		{ParseIDR(7500), "Rp 7.500"},
	}
	for _, c := range cases {
		if got := c.in.String(); got != c.want {
			t.Errorf("IDR(%d).String() = %q, want %q", c.in, got, c.want)
		}
	}
}
