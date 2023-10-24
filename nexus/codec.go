package nexus

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type Stream struct {
	Header map[string]string
	// TODO: should it be a closer too?
	Reader io.Reader
}

type EncodedStream struct {
	codec  Codec
	stream *Stream
}

func (s *EncodedStream) Read(v any) error {
	defer s.Close()
	return s.codec.Deserialize(s.stream, v)
}

func (s *EncodedStream) Close() error {
	if closer, ok := s.stream.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

type Codec interface {
	Serialize(any) (*Stream, error)
	Deserialize(*Stream, any) error
}

type CodecChain []Codec

var ErrCodecIncompatible = errors.New("incompatible codec")

func (c CodecChain) Serialize(v any) (*Stream, error) {
	for _, l := range c {
		p, err := l.Serialize(v)
		if err != nil {
			if errors.Is(err, ErrCodecIncompatible) {
				continue
			}
			return nil, err
		}
		return p, nil
	}
	return nil, ErrCodecIncompatible
}

func (c CodecChain) Deserialize(s *Stream, v any) error {
	lenc := len(c)
	for i := range c {
		l := c[lenc-i-1]
		if err := l.Deserialize(s, v); err != nil {
			if errors.Is(err, ErrCodecIncompatible) {
				continue
			}
			return err
		}
		return nil
	}
	return ErrCodecIncompatible
}

var _ Codec = CodecChain{}

type JSONCodec struct{}

func (JSONCodec) Deserialize(s *Stream, v any) error {
	// TODO: isContentTypeJSON(s.Header)
	if s.Header["Content-Type"] != "application/json" {
		return ErrCodecIncompatible
	}
	body, err := io.ReadAll(s.Reader)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, &v)
}

func (JSONCodec) Serialize(v any) (*Stream, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &Stream{
		Header: map[string]string{
			"Content-Type":   "application/json",
			"Content-Length": fmt.Sprintf("%d", len(body)),
		},
		Reader: bytes.NewReader(body),
	}, nil
}

var _ Codec = JSONCodec{}

type NilCodec struct{}

func (NilCodec) Deserialize(s *Stream, v any) error {
	// TODO: case sensitivity
	if s.Header["Content-Length"] == "0" {
		return ErrCodecIncompatible
	}
	return nil
}

func (NilCodec) Serialize(v any) (*Stream, error) {
	if v != nil {
		return nil, ErrCodecIncompatible
	}
	return &Stream{
		Header: map[string]string{"Content-Length": "0"},
		Reader: nil,
	}, nil
}

var _ Codec = NilCodec{}

type ByteSliceCodec struct{}

func (ByteSliceCodec) Deserialize(s *Stream, v any) error {
	// TODO: media type
	if s.Header["Content-Type"] != "application/octet-stream" {
		return ErrCodecIncompatible
	}
	if bRef, ok := v.(*[]byte); ok {
		b, err := io.ReadAll(s.Reader)
		if err != nil {
			return err
		}
		*bRef = b
		return nil
	}
	return errors.New("unsupported value type for content")
}

func (ByteSliceCodec) Serialize(v any) (*Stream, error) {
	if b, ok := v.([]byte); ok {
		return &Stream{
			Header: map[string]string{
				"Content-Type":   "application/octet-stream",
				"Content-Length": fmt.Sprintf("%d", len(b)),
			},
			Reader: bytes.NewReader(b),
		}, nil
	}
	return nil, ErrCodecIncompatible
}

var _ Codec = NilCodec{}

var DefaultCodec = CodecChain([]Codec{NilCodec{}, ByteSliceCodec{}, JSONCodec{}})
