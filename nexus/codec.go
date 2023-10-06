package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var ErrEncodingNotSupported = errors.New("encoding not supported")

type Message struct {
	Header http.Header
	Body   io.Reader
}

type MessageDecoder interface {
	FromMessage(*Message, any) error
}

type MessageEncoder interface {
	ToMessage(any) (*Message, error)
}

type MessageCodec interface {
	MessageEncoder
	MessageDecoder
}

type FailureCodec interface {
	FromFailure(*Failure) (error, error)
	ToFailure(error) (*Failure, error)
}

type ByteCodec interface {
	EncodeBytes(*Message) (*Message, error)
	DecodeBytes(*Message) (*Message, error)
}

type Codec interface {
	MessageCodec
	FailureCodec
	ByteCodec
}

type JSONCodec struct{}

// Encode implements Codec.
func (JSONCodec) FromMessage(m *Message, v any) error {
	if !isContentTypeJSON(m.Header) {
		return ErrEncodingNotSupported
	}
	return json.NewDecoder(m.Body).Decode(v)
}

// Decode implements Codec.
func (JSONCodec) ToMessage(v any) (*Message, error) {
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(v)
	if err != nil {
		return nil, err
	}
	return &Message{
		Header: http.Header{"Content-Type": []string{"application/json"}},
		Body:   body,
	}, nil
}

var _ MessageCodec = JSONCodec{}

type DefaultErrorCodec struct{}

// DecodeError implements ErrorCodec.
func (DefaultErrorCodec) FromFailure(failure *Failure) (error, error) {
	return errors.New(failure.Message), nil
}

// EncodeError implements ErrorCodec.
func (DefaultErrorCodec) ToFailure(err error) (*Failure, error) {
	return &Failure{Message: err.Error()}, nil
}

type DefaultByteCodec struct{}

// EncodeBytes implements ByteCodec.
func (DefaultErrorCodec) EncodeBytes(m *Message) (*Message, error) {
	return m, nil
}

// DecodeBytes implements ByteCodec.
func (DefaultErrorCodec) DecodeBytes(m *Message) (*Message, error) {
	return m, nil
}

type DefaultCodec struct {
	DefaultErrorCodec
	JSONCodec
	DefaultByteCodec
}

type contextKeyCodec struct{}

func GetCodec(ctx context.Context) Codec {
	return ctx.Value(contextKeyCodec{}).(Codec)
}

func WithCodec(ctx context.Context, codec Codec) context.Context {
	return context.WithValue(ctx, contextKeyCodec{}, codec)
}

type ByteCodecChain []ByteCodec

func (c ByteCodecChain) EncodeBytes(m *Message) (*Message, error) {
	for _, l := range c {
		var err error
		m, err = l.EncodeBytes(m)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (c ByteCodecChain) DecodeBytes(m *Message) (*Message, error) {
	lenc := len(c)
	for i := range c {
		l := c[lenc-i-1]
		var err error
		m, err = l.DecodeBytes(m)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}
