package jet

const largestValueSize = int64(^uint64(0) >> 1)

func addValueSizes(a, b int64) (int64, bool) {
	if a < 0 || b < 0 || b > largestValueSize-a {
		return 0, false
	}
	return a + b, true
}

func multiplyValueSize(count, width int64) (int64, bool) {
	if count < 0 || width < 0 || (width != 0 && count > largestValueSize/width) {
		return 0, false
	}
	return count * width, true
}

func replacedValueSize(source, old, replacement, matches int64) (int64, bool) {
	removed, ok := multiplyValueSize(old, matches)
	if !ok || source < removed {
		return 0, false
	}
	added, ok := multiplyValueSize(replacement, matches)
	if !ok {
		return 0, false
	}
	return addValueSizes(source-removed, added)
}
