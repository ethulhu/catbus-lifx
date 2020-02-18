package lifx

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"time"
)

var (
	source = rand.Uint32()

	broadcastAddr = &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: 56700,
	}
)

// Discover discovers Bulbs within a given timeout.
func Discover(ctx context.Context) (<-chan Bulb, error) {
	packet := broadcastPacket(&getService{})

	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on UDP: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	if _, err := conn.WriteTo(packet, broadcastAddr); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send packet on UDP: %w", err)
	}

	bulbs := make(chan Bulb)
	returned := make(chan struct{})

	go func() {
		select {
		case <-returned:
			// do nothing.
		case <-ctx.Done():
			conn.SetDeadline(time.Now())
		}
	}()

	go func() {
		defer conn.Close()
		defer close(bulbs)
		defer close(returned)

		bulbIDs := map[uint64]bool{}
		for {
			buf := make([]byte, 256)
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}

			hdr := &header{}
			hdr.FromBytes(buf[0:headerLength])

			// If we've already seen it, skip.
			if bulbIDs[hdr.Target] {
				continue
			}
			bulbIDs[hdr.Target] = true

			message := &stateService{}
			reader := bytes.NewReader(buf[headerLength:n])
			_ = binary.Read(reader, binary.LittleEndian, message)

			bulb := &bulb{
				addr: &net.UDPAddr{
					IP:   addr.IP,
					Port: int(message.Port),
				},
				id: hdr.Target,
			}

			bulbs <- bulb
		}
	}()

	return bulbs, nil
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
