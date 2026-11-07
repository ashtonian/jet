package jet

import (
	"strings"
	"testing"
)

func TestValueSizeArithmetic(t *testing.T) {
	for _, tt := range []struct {
		name string
		size func(int64, int64) (int64, bool)
		a, b int64
		want int64
		ok   bool
	}{
		{"sum", addValueSizes, 7, 9, 16, true},
		{"sumBoundary", addValueSizes, largestValueSize - 1, 1, largestValueSize, true},
		{"sumOverflow", addValueSizes, largestValueSize, 1, 0, false},
		{"sumNegative", addValueSizes, -1, 1, 0, false},
		{"product", multiplyValueSize, 7, 9, 63, true},
		{"productBoundary", multiplyValueSize, largestValueSize, 1, largestValueSize, true},
		{"productOverflow", multiplyValueSize, largestValueSize/2 + 1, 2, 0, false},
		{"productNegative", multiplyValueSize, 1, -1, 0, false},
		{"productZero", multiplyValueSize, largestValueSize, 0, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.size(tt.a, tt.b)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("got (%d, %t), want (%d, %t)", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestReplacedValueSize(t *testing.T) {
	for _, tt := range []struct {
		name, source, old, replacement string
		count                          int
	}{
		{"grow", "ababa", "a", "xyz", -1},
		{"shrink", "ababa", "ab", "x", -1},
		{"limited", "ababa", "a", "xyz", 1},
		{"emptyOld", "äb", "", "xyz", -1},
		{"emptySource", "", "", "xyz", -1},
		{"delete", "ababa", "a", "", -1},
		{"noMatch", "ababa", "c", "xyz", -1},
		{"noReplacement", "ababa", "a", "xyz", 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			matches := strings.Count(tt.source, tt.old)
			if tt.count >= 0 && matches > tt.count {
				matches = tt.count
			}
			got, ok := replacedValueSize(int64(len(tt.source)), int64(len(tt.old)), int64(len(tt.replacement)), int64(matches))
			want := int64(len(strings.Replace(tt.source, tt.old, tt.replacement, tt.count)))
			if !ok || got != want {
				t.Fatalf("got (%d, %t), want (%d, true)", got, ok, want)
			}
		})
	}
	for _, tt := range []struct {
		name                              string
		source, old, replacement, matches int64
	}{
		{"sumOverflow", largestValueSize, 0, 1, 1},
		{"productOverflow", 1, 0, largestValueSize, 2},
		{"removedOverflow", largestValueSize, largestValueSize, 0, 2},
		{"excessMatches", 2, 1, 0, 3},
		{"negativeSource", -1, 0, 1, 1},
		{"negativeMatches", 1, 0, 1, -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got, ok := replacedValueSize(tt.source, tt.old, tt.replacement, tt.matches); ok {
				t.Fatalf("accepted invalid construction size %d", got)
			}
		})
	}
}
