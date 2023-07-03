package utils

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"
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

func ReadMessage(conn net.Conn) (Message, error) {
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	msgLength := make([]byte, 4)
	if _, err := io.ReadFull(conn, msgLength); err != nil {
		// log.Print("Error reading message length: ", err)
		return Message{}, err
	}
	length := binary.BigEndian.Uint32(msgLength)

	msgId := make([]byte, 1)
	if _, err := io.ReadFull(conn, msgId); err != nil {
		// log.Print("Error reading message ID: ", err)
		return Message{}, err
	}
	id := messageID(msgId[0])

	if id == MsgChoke || id == MsgUnchoke {
		time.Sleep(time.Second)
		return ReadMessage(conn)
	}

	if id > 8 {
		return Message{}, errors.New("invalid message ID")
	}

	// log.Print("Received message with length ", length, " and ID ", id)
	buffer := make([]byte, length-1)
	if _, err := io.ReadFull(conn, buffer); err != nil {
		// log.Print("Error reading message body: ", err)
		return Message{}, err
	}

	return Message{ID: messageID(id), Payload: buffer}, nil
}

func RequestMessage(index int, offset int, blockSize int) Message {
	requestPayload := make([]byte, 12)
	binary.BigEndian.PutUint32(requestPayload[0:4], uint32(index))
	binary.BigEndian.PutUint32(requestPayload[4:8], uint32(offset))
	binary.BigEndian.PutUint32(requestPayload[8:12], uint32(blockSize))

	return Message{ID: MsgRequest, Payload: requestPayload}
}

func InterestedMessage() Message {
	return Message{ID: MsgInterested, Payload: make([]byte, 0)}
}
