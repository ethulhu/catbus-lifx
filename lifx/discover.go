package lifx

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
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

	// TODO: provide a proper listenAddr.
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %w", err)
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	}

	if _, err := conn.WriteTo(packet, broadcastAddr); err != nil {
		return nil, fmt.Errorf("failed to send packet on UDP: %w", err)
	}

	var bulbs []Bulb
	bulbIDs := map[uint64]bool{}
	for {
		buf := make([]byte, 256)
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			var netError net.Error
			if errors.As(err, &netError) && netError.Timeout() {
				return bulbs, nil
			}
			return bulbs, fmt.Errorf("failed to read from UDP: %w", err)
		}

		hdr := &header{}
		hdr.FromBytes(buf[0:headerLength])

		// If we've already seen it, skip.
		if bulbIDs[hdr.Target] {
			continue
		}
		bulbIDs[hdr.Target] = true

		message := &stateService{}
		expectedType := typeForMessage(message)
		if hdr.Type != expectedType {
			return bulbs, fmt.Errorf("unexpected message type: expected %v, got %v", expectedType, hdr.Type)
		}

		reader := bytes.NewReader(buf[headerLength:n])
		_ = binary.Read(reader, binary.LittleEndian, message)

		bulb := &bulb{
			addr: &net.UDPAddr{
				IP:   addr.IP,
				Port: int(message.Port),
			},
			id: hdr.Target,
		}

		bulbs = append(bulbs, bulb)
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
