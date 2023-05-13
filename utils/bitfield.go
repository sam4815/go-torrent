package utils

type Bitfield []byte

func CreateBitfield(length int) Bitfield {
	bitfieldLength := 1 + (length+7)/8
	return make([]byte, bitfieldLength)
}

func (b Bitfield) HasPiece(index int) bool {
	targetByte := b[index/8]
	bitOffset := index % 8

	shifted := targetByte >> (7 - bitOffset)
	smallestBit := shifted & 1

	return smallestBit == 1
}

func (b *Bitfield) SetPiece(index int) {
	bitOffset := index % 8

	andable := byte(2 ^ (7 - bitOffset))
	(*b)[index/8] &= andable
}
