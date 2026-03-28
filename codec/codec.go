package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	// MaxMessageSize sets a limit on the maximum allowed message payload size
	// to prevent Out-Of-Memory issues from maliciously or incorrectly framed data.
	MaxMessageSize = 10 * 1024 * 1024 // 10 MB
)

var (
	ErrInvalidPreamble = errors.New("invalid frame preamble, expected 0x00")
	ErrMessageTooLarge = errors.New("message size exceeds maximum allowed")
)

// WriteFrame writes an ESPHome frame to w.
// Format: 0x00 | varint(msg size) | varint(msg type) | data
func WriteFrame(w io.Writer, msgType uint32, data []byte) error {
	// Write preamble
	if _, err := w.Write([]byte{0x00}); err != nil {
		return fmt.Errorf("failed to write preamble: %w", err)
	}

	var buf [10]byte
	// Write size
	sizeLen := binary.PutUvarint(buf[:], uint64(len(data)))
	if _, err := w.Write(buf[:sizeLen]); err != nil {
		return fmt.Errorf("failed to write size: %w", err)
	}

	// Write type
	typeLen := binary.PutUvarint(buf[:], uint64(msgType))
	if _, err := w.Write(buf[:typeLen]); err != nil {
		return fmt.Errorf("failed to write message type: %w", err)
	}

	// Write payload
	if len(data) > 0 {
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("failed to write payload: %w", err)
		}
	}

	return nil
}

// readByteReader wraps an io.Reader to provide io.ByteReader.
type readByteReader struct {
	io.Reader
	buf [1]byte
}

func (r *readByteReader) ReadByte() (byte, error) {
	n, err := r.Read(r.buf[:])
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, io.EOF
	}
	return r.buf[0], nil
}

// ReadFrame reads an ESPHome frame from r.
func ReadFrame(r io.Reader) (msgType uint32, data []byte, err error) {
	br, ok := r.(io.ByteReader)
	if !ok {
		br = &readByteReader{Reader: r}
	}

	// Read preamble
	preamble, err := br.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	if preamble != 0x00 {
		return 0, nil, ErrInvalidPreamble
	}

	// Read size
	size, err := binary.ReadUvarint(br)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read size: %w", err)
	}

	if size > MaxMessageSize {
		return 0, nil, ErrMessageTooLarge
	}

	// Read type
	mType, err := binary.ReadUvarint(br)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read message type: %w", err)
	}
	msgType = uint32(mType)

	// Read payload
	if size > 0 {
		data = make([]byte, size)
		_, err = io.ReadFull(r, data)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to read payload: %w", err)
		}
	} else {
		data = nil
	}

	return msgType, data, nil
}
