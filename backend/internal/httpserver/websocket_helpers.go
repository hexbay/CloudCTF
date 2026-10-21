package httpserver

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

func websocketAcceptKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	sum := sha1.Sum([]byte(trimmed + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func readWebSocketFrame(reader *bufio.Reader) (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return 0, nil, err
	}

	opcode := header[0] & 0x0F
	masked := header[1]&0x80 != 0
	payloadLen := uint64(header[1] & 0x7F)

	switch payloadLen {
	case 126:
		extended := make([]byte, 2)
		if _, err := io.ReadFull(reader, extended); err != nil {
			return 0, nil, err
		}
		payloadLen = uint64(binary.BigEndian.Uint16(extended))
	case 127:
		extended := make([]byte, 8)
		if _, err := io.ReadFull(reader, extended); err != nil {
			return 0, nil, err
		}
		payloadLen = binary.BigEndian.Uint64(extended)
	}

	maskKey := make([]byte, 4)
	if masked {
		if _, err := io.ReadFull(reader, maskKey); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return 0, nil, err
	}

	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}
	return opcode, payload, nil
}

func writeWebSocketTextFrame(writer io.Writer, payload []byte) error {
	if len(payload) > 65535 {
		return fmt.Errorf("websocket payload too large")
	}

	header := []byte{0x81}
	if len(payload) < 126 {
		header = append(header, byte(len(payload)))
	} else {
		header = append(header, 126, byte(len(payload)>>8), byte(len(payload)))
	}

	if _, err := writer.Write(header); err != nil {
		return err
	}
	_, err := writer.Write(payload)
	return err
}
