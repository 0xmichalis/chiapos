package pos

// Match is a matching function.
func Match(left, right uint64) bool {
	if BucketID(left)+1 != BucketID(right) {
		return false
	}
	bIDLeft, cIDLeft := GetIDs(left)
	bIDRight, cIDRight := GetIDs(right)
	for m := uint64(0); m < paramM; m++ {
		firstCondition := (bIDRight-bIDLeft)%paramB == m%paramB
		secondCondition := (cIDRight-cIDLeft)%paramC == (2*m+(BucketID(left)%2))^2%paramC
		if firstCondition && secondCondition {
			return true
		}
	}
	return false
}
