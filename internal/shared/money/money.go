// Package money models IDR amounts for permit/cost fields. Amounts are stored
// and transported as whole rupiah (int64); the API serialises them as JSON
// numbers and the frontend formats with Intl.NumberFormat (IDR) for display.
package money

import "fmt"

// IDR is a whole-rupiah amount.
type IDR int64

// Zero is the zero amount.
const Zero IDR = 0

// ParseIDR converts a whole-rupiah int64 into an IDR amount.
func ParseIDR(v int64) IDR { return IDR(v) }

// String renders Indonesian thousand separators, e.g. "Rp 250.000".
func (m IDR) String() string {
	return "Rp " + formatThousands(int64(m))
}

// Int64 returns the whole-rupiah value for JSON serialisation.
func (m IDR) Int64() int64 { return int64(m) }

func formatThousands(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	// Insert '.' every three digits from the right.
	out := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}
