package utils

import (
	"encoding/binary"
	"io"
	"log"
	"net"
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

func ToMessage(conn net.Conn) Message {
	msgLength := make([]byte, 4)
	if _, err := io.ReadFull(conn, msgLength); err != nil {
		log.Print(err)
		return Message{}
	}
	length := binary.BigEndian.Uint32(msgLength)

	if length == 0 {
		return Message{}
	}

	log.Print("RESPONSE LENGTH: ", length)

	msgId := make([]byte, 1)
	if _, err := io.ReadFull(conn, msgId); err != nil {
		log.Print(err)
		return Message{}
	}
	id := msgId[0]
	log.Print("RESPONSE ID: ", id)

	if id > 8 {
		log.Print("INVALID RESPONSE")
		return Message{}
	}

	buffer := make([]byte, length)
	io.ReadFull(conn, buffer)

	return Message{ID: messageID(id), Payload: buffer}
}

func RequestMessage(index int, offset int, blockSize int) Message {
	requestPayload := make([]byte, 12)
	binary.BigEndian.PutUint32(requestPayload[0:4], uint32(index))
	binary.BigEndian.PutUint32(requestPayload[4:8], uint32(offset))
	binary.BigEndian.PutUint32(requestPayload[8:12], uint32(blockSize)) // 16KB

	return Message{ID: MsgRequest, Payload: requestPayload}
}
