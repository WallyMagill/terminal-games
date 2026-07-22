package engine

const Size = 4

// slideLine slides and merges toward index 0 (left).
// Equal adjacent tiles merge once; a merged tile does not merge again.
// scoreDelta is the sum of values of tiles created by merges.
func slideLine(line [Size]int) (out [Size]int, scoreDelta int) {
	n := 0
	var compact [Size]int
	for i := 0; i < Size; i++ {
		if line[i] != 0 {
			compact[n] = line[i]
			n++
		}
	}
	var merged [Size]int
	m := 0
	for i := 0; i < n; {
		if i+1 < n && compact[i] == compact[i+1] {
			v := compact[i] * 2
			merged[m] = v
			scoreDelta += v
			m++
			i += 2
			continue
		}
		merged[m] = compact[i]
		m++
		i++
	}
	return merged, scoreDelta
}
