package codec

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/iotest"
)

func TestFrameRoundTrip(t *testing.T) {
	msgType := uint32(42)
	payload := []byte("hello ESPHome")

	var buf bytes.Buffer
	err := WriteFrame(&buf, msgType, payload)
	if err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	decodedType, decodedData, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	if decodedType != msgType {
		t.Errorf("expected msgType %d, got %d", msgType, decodedType)
	}
	if !bytes.Equal(decodedData, payload) {
		t.Errorf("expected payload %q, got %q", payload, decodedData)
	}
}

func TestPartialReads(t *testing.T) {
	msgType := uint32(99)
	payload := []byte("a very long payload that we will read partially in this test")

	var buf bytes.Buffer
	err := WriteFrame(&buf, msgType, payload)
	if err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	// use iotest.HalfReader to simulate partial network reads
	hr := iotest.HalfReader(&buf)

	decodedType, decodedData, err := ReadFrame(hr)
	if err != nil {
		t.Fatalf("ReadFrame with HalfReader failed: %v", err)
	}

	if decodedType != msgType {
		t.Errorf("expected msgType %d, got %d", msgType, decodedType)
	}
	if !bytes.Equal(decodedData, payload) {
		t.Errorf("expected payload %q, got %q", payload, decodedData)
	}
}

func TestInvalidFrames(t *testing.T) {
	t.Run("missing zero byte", func(t *testing.T) {
		buf := bytes.NewBuffer([]byte{0x01, 0x05, 0x01, 0x02, 0x03})
		_, _, err := ReadFrame(buf)
		if !errors.Is(err, ErrInvalidPreamble) {
			t.Errorf("expected ErrInvalidPreamble, got %v", err)
		}
	})

	t.Run("truncated varint", func(t *testing.T) {
		// Valid preamble but no varint data
		buf := bytes.NewBuffer([]byte{0x00})
		_, _, err := ReadFrame(buf)
		if err == nil {
			t.Error("expected error on truncated varint")
		}
	})

	t.Run("size exceeding limit", func(t *testing.T) {
		// write preamble
		var buf bytes.Buffer
		buf.WriteByte(0x00)
		// write large size varint (larger than MaxMessageSize)
		// 11 MB > 10 MB
		buf.Write([]byte{0x80, 0x80, 0x80, 0x08})

		_, _, err := ReadFrame(&buf)
		if !errors.Is(err, ErrMessageTooLarge) {
			t.Errorf("expected ErrMessageTooLarge, got %v", err)
		}
	})
}

func TestMultipleFrames(t *testing.T) {
	frames := []struct {
		msgType uint32
		data    []byte
	}{
		{msgType: 1, data: []byte("first")},
		{msgType: 2, data: []byte("second")},
		{msgType: 0, data: []byte("")}, // zero-length payload
		{msgType: 100, data: []byte("last")},
	}

	var buf bytes.Buffer
	for _, f := range frames {
		if err := WriteFrame(&buf, f.msgType, f.data); err != nil {
			t.Fatalf("WriteFrame failed: %v", err)
		}
	}

	for i, f := range frames {
		mType, mData, err := ReadFrame(&buf)
		if err != nil {
			t.Fatalf("ReadFrame %d failed: %v", i, err)
		}
		if mType != f.msgType {
			t.Errorf("frame %d: expected type %d, got %d", i, f.msgType, mType)
		}
		if !bytes.Equal(mData, f.data) {
			t.Errorf("frame %d: expected data %q, got %q", i, f.data, mData)
		}
	}

	// Should be EOF now
	_, _, err := ReadFrame(&buf)
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected EOF at end of buffer, got %v", err)
	}
}
