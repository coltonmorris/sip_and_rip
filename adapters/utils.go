package adapters

import (
	"crypto/rand"
)

func RandUint32() uint32 {
	// 4 bytes = 32 bits
	var b [4]byte

	// generate 4 random bytes
	rand.Read(b[:])

	// combine the 4 bytes into a single uint32
	// when combining using the | operator, the bits are shifted left to line up correctly
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

func RandUint16() uint16 {
	// 2 bytes = 16 bits
	var b [2]byte

	// generate 2 random bytes
	rand.Read(b[:])

	// combine the 2 bytes into a single uint16
	// when combining using the | operator, the bits are shifted left to line up correctly
	return uint16(b[0])<<8 | uint16(b[1])
}
