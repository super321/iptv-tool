package utils

import (
	"strconv"
	"unicode"
)

// NaturalLess compares two strings in natural sort order.
// e.g. "CCTV-2" < "CCTV-10"
func NaturalLess(str1, str2 string) bool {
	r1 := []rune(str1)
	r2 := []rune(str2)

	i1, i2 := 0, 0
	l1, l2 := len(r1), len(r2)

	for i1 < l1 && i2 < l2 {
		ch1 := r1[i1]
		ch2 := r2[i2]

		isDigit1 := unicode.IsDigit(ch1)
		isDigit2 := unicode.IsDigit(ch2)

		// Both are numeric, process the number chunk
		if isDigit1 && isDigit2 {
			j1, j2 := i1, i2
			for j1 < l1 && unicode.IsDigit(r1[j1]) {
				j1++
			}
			for j2 < l2 && unicode.IsDigit(r2[j2]) {
				j2++
			}

			num1, err1 := strconv.ParseUint(string(r1[i1:j1]), 10, 64)
			num2, err2 := strconv.ParseUint(string(r2[i2:j2]), 10, 64)

			if err1 == nil && err2 == nil {
				if num1 != num2 {
					return num1 < num2
				}
				// If numeric values are equal (e.g. "01" vs "1"), sort by string length
				if j1-i1 != j2-i2 {
					return j1-i1 < j2-i2
				}
			} else {
				// Fallback to string comparison if too large
				sPart1 := string(r1[i1:j1])
				sPart2 := string(r2[i2:j2])
				if sPart1 != sPart2 {
					return sPart1 < sPart2
				}
			}

			i1, i2 = j1, j2
			continue
		}

		// Not both are numeric, check character by character
		if ch1 != ch2 {
			return ch1 < ch2
		}

		i1++
		i2++
	}

	// Shorter string comes first if all previous match
	return l1 < l2
}
