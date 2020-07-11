// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package lifx

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"
)

type (
	bulb struct {
		id   uint64
		addr net.Addr

		sequence uint8
		sync.Mutex
	}
)

func (b *bulb) String() string {
	return fmt.Sprintf("{ id: %v addr: %v }", b.id, b.addr)
}

func (b *bulb) State(ctx context.Context) (State, error) {
	m, err := b.sendAndReceive(ctx, &get{})
	if err != nil {
		return State{}, err
	}

	rawState, ok := m.(*state)
	if !ok {
		return State{}, fmt.Errorf("expected State message, got message type %v", reflect.TypeOf(m))
	}
	return prettyState(rawState), nil
}

func (b *bulb) SetColor(ctx context.Context, hsbk HSBK, d time.Duration) error {
	color, err := uglyHSBK(hsbk)
	if err != nil {
		return err
	}
	req := &setColor{
		Color:    color,
		Duration: uint32(d.Milliseconds()),
	}

	_, err = b.sendAndReceive(ctx, req)
	return err
}

func (b *bulb) SetPower(ctx context.Context, p Power, d time.Duration) error {
	req := &setPower{
		Power:    uint16(p),
		Duration: uint32(d.Milliseconds()),
	}
	_, err := b.sendAndReceive(ctx, req)
	return err
}
func (b *bulb) sendAndReceive(ctx context.Context, message interface{}) (interface{}, error) {
	var payload bytes.Buffer
	_ = binary.Write(&payload, binary.LittleEndian, message)

	hdr := &header{
		Size:             uint16(headerLength + payload.Len()),
		Tagged:           false,
		Source:           source,
		Target:           b.id,
		ResponseRequired: true,
		Sequence:         b.nextSequence(),
		Type:             typeForMessage(message),
	}
	packet := append(hdr.Bytes(), payload.Bytes()...)

	conn, err := net.Dial(b.addr.Network(), b.addr.String())
	if err != nil {
		return nil, fmt.Errorf("could not dial bulb: %w", err)
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	}

	if _, err := conn.Write(packet); err != nil {
		return nil, fmt.Errorf("could not send packet: %w", err)
	}

	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil {
		var netError net.Error
		if errors.As(err, &netError) && netError.Timeout() {
			return nil, ErrNoResponse
		}
		return nil, fmt.Errorf("could not read packet: %w", err)
	}

	hdr = &header{}
	hdr.FromBytes(buf[0:headerLength])

	message = messageForType(hdr.Type)
	reader := bytes.NewReader(buf[headerLength:n])
	_ = binary.Read(reader, binary.LittleEndian, message)
	return message, nil
}

func (b *bulb) nextSequence() uint8 {
	b.Lock()
	defer b.Unlock()

	seq := b.sequence
	if seq == 255 {
		b.sequence = 0
	} else {
		b.sequence++
	}
	return seq
}
