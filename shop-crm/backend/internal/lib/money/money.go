package money

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// KopPerRuble is the number of kopecks in one ruble. All monetary values in the
// system are integers of kopecks; float arithmetic is never used for money.
const KopPerRuble = 100

var ErrNegative = errors.New("money: negative amount")
var ErrTooPrecise = errors.New("money: more than two decimal places")

// RubToKop converts a ruble amount to kopecks without float rounding.
// "12.34" -> 1234.
func RubToKop(s string) (int64, error) {
	var whole int64
	var frac int
	var digits int
	var seenDot bool
	var gotDigit bool

	for i, r := range s {
		switch {
		case r == ',' || r == '.':
			// A separator is only allowed between digits: no "12.", no ".50".
			if seenDot || !gotDigit {
				return 0, errors.New("money: invalid amount " + s)
			}
			seenDot = true
			gotDigit = false
		case r == '-' && i == 0:
			return 0, ErrNegative
		case r >= '0' && r <= '9':
			gotDigit = true
			if seenDot {
				digits++
				if digits > 2 {
					return 0, ErrTooPrecise
				}
				frac = frac*10 + int(r-'0')
			} else {
				whole = whole*10 + int64(r-'0')
				if whole > math.MaxInt32 {
					return 0, errors.New("money: amount too large " + s)
				}
			}
		default:
			return 0, errors.New("money: invalid amount " + s)
		}
	}
	if !gotDigit {
		return 0, errors.New("money: invalid amount " + s)
	}
	if digits == 1 {
		frac *= 10
	}
	return whole*KopPerRuble + int64(frac), nil
}

// FormatKop renders kopecks the way the till shows them: "1 234 ₽", with the kopeck
// part added as ",50" when it is not zero. Negative values keep the sign in front.
func FormatKop(kop int64) string {
	negative := kop < 0
	if negative {
		kop = -kop
	}

	rubles := kop / KopPerRuble
	fraction := kop % KopPerRuble

	out := groupThousands(rubles)
	if fraction != 0 {
		out += "," + fmt.Sprintf("%02d", fraction)
	}
	out += " ₽"
	if negative {
		out = "-" + out
	}
	return out
}

func groupThousands(v int64) string {
	s := fmt.Sprintf("%d", v)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	// A narrow no-break space keeps "1 234 ₽" on one line in every browser.
	return strings.Join(parts, "\u202f")
}