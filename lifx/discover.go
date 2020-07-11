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
	"io"
	"math/rand"
	"net"
)

var (
	source = rand.Uint32()

	broadcastAddr = &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: 56700,
	}
)

// Discover discovers Bulbs within a given timeout.
func Discover(ctx context.Context) ([]Bulb, error) {
	packet := broadcastPacket(&getService{})

	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("could not listen on UDP: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}
	defer conn.Close()

	if _, err := conn.WriteTo(packet, broadcastAddr); err != nil {
		return nil, fmt.Errorf("could not send discover packet: %w", err)
	}

	bulbIDs := map[uint64]bool{}
	var bulbs []Bulb
	for {
		buf := make([]byte, 256)
		n, addr, err := conn.ReadFromUDP(buf)

		var netErr net.Error
		if errors.Is(err, io.EOF) || (errors.As(err, &netErr) && netErr.Timeout()) {
			return bulbs, nil
		}
		if err != nil {
			return bulbs, err
		}

		hdr := &header{}
		hdr.FromBytes(buf[0:headerLength])

		// If we've already seen it, skip.
		if bulbIDs[hdr.Target] {
			continue
		}

		message := &stateService{}
		reader := bytes.NewReader(buf[headerLength:n])
		_ = binary.Read(reader, binary.LittleEndian, message)

		bulbIDs[hdr.Target] = true
		bulbs = append(bulbs, &bulb{
			addr: &net.UDPAddr{
				IP:   addr.IP,
				Port: int(message.Port),
			},
			id: hdr.Target,
		})
	}
}

func broadcastPacket(message interface{}) []byte {
	var payload bytes.Buffer
	_ = binary.Write(&payload, binary.LittleEndian, message)

	hdr := &header{
		Size:     uint16(headerLength + payload.Len()),
		Tagged:   true,
		Source:   source,
		Target:   uint64(0),
		Sequence: 0,
		Type:     typeForMessage(message),
	}

	return append(hdr.Bytes(), payload.Bytes()...)
}
