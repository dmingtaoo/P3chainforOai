package utils

import (
	"fmt"
)

// convert a bool slice into int64, the length could
// not over 64.
func BoolByteSliceToUInt64(bbs []byte) (uint64, error) {
	if len(bbs) > 64 {
		return 0, fmt.Errorf("bool list length over")
	}
	resInt := uint64(0)

	for _, ele := range bbs {
		if ele == byte(1) {
			resInt = resInt<<1 + 1
		} else {
			resInt = resInt << 1
		}
	}

	return resInt, nil
}

// convert a bool slice into string
func BoolByteSliceToString(bbs []byte) string {
	return string(bbs)
}
