package utils

import (
	"encoding/binary"
	"io"
)

type messageID uint8

const (
	MsgChoke        messageID = 0
	MsgUnchoke      messageID = 1
	MsgInterested   messageID = 2
	MsgUninterested messageID = 3
	MsgHave         messageID = 4
	MsgBitfield     messageID = 5
	MsgRequest      messageID = 6
	MsgPiece        messageID = 7
	MsgCancel       messageID = 8
)

type Message struct {
	ID      messageID
	Payload []byte
}

func (m Message) ToBytes() []byte {
	buffer := make([]byte, len(m.Payload)+5)

	binary.BigEndian.PutUint32(buffer[0:4], uint32(len(m.Payload)+1)) // message length
	buffer[4] = byte(m.ID)                                            // message ID
	copy(buffer[5:], m.Payload)                                       // message payload

	return buffer
}

func ToMessage(r io.Reader) Message {
	buffer := make([]byte, 4)
	io.ReadFull(r, buffer)
	length := binary.BigEndian.Uint32(buffer[0:4])

	if length == 0 {
		return Message{}
	}

	buffer = make([]byte, length)
	io.ReadFull(r, buffer)

	return Message{ID: messageID(buffer[0]), Payload: buffer[1:]}
}

func RequestMessage(index int, offset int, blockSize int) Message {
	requestPayload := make([]byte, 12)
	binary.BigEndian.PutUint32(requestPayload[0:4], uint32(index))
	binary.BigEndian.PutUint32(requestPayload[4:8], uint32(offset))
	binary.BigEndian.PutUint32(requestPayload[8:12], uint32(blockSize)) // 16KB

	return Message{ID: MsgRequest, Payload: requestPayload}
}
