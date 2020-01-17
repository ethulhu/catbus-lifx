package lifx

import (
	"encoding/binary"
	"fmt"
)

const headerLength = 36

// header is the useful / usable fields from a Lifx LAN protocol header.
//
// From the Lifx LAN documentation at https://github.com/LIFX/lifx-protocol-docs:
//
// 	typedef struct {
// 		// frame
// 		uint16_t size;
// 		uint16_t protocol:12; // Must be 1024.
// 		uint8_t  addressable:1;
// 		uint8_t  tagged:1;
// 		uint8_t  origin:2;
// 		uint32_t source;
// 		// frame address
// 		uint8_t  target[8];
// 		uint8_t  reserved[6];
// 		uint8_t  res_required:1;
// 		uint8_t  ack_required:1;
// 		uint8_t  :6;
// 		uint8_t  sequence;
// 		// protocol header
// 		uint64_t :64;
// 		uint16_t type;
// 		uint16_t :16;
// 		// variable length payload follows
// 	} lx_protocol_header_t;
type header struct {
	// Frame.

	// Size is the size of the entire message in bytes.
	Size uint16
	// Tagged specifies if Target is to address a specific bulb or all bulbs.
	// TODO: fill in the rest of the details.
	Tagged bool
	// Source is a unique identifier for the client.
	// If Source is 0, the bulb may send a broadcast message.
	// Otherwise, the bulb will reply directly to the client.
	Source uint32

	// Frame Address.

	// Target is the MAC address of the bulb being queried.
	// For Broadcast requests, e.g. discovery, Target should be all zeros.
	Target uint64

	// ResponseRequired determines whether the bulb should respond, e.g. to SetPower with StatePower.
	ResponseRequired bool

	// AcknowledgementRequired determines whether the bulb should respond with an Acknlowledgement message.
	AcknowledgementRequired bool

	// Sequence is an incrementing sequence number.
	Sequence uint8

	// Protocol Header.

	// Type is the message Type, and determines the payload.
	Type uint16
}

func (h *header) Bytes() []byte {
	data := make([]byte, headerLength)

	// Frame.

	binary.LittleEndian.PutUint16(data[0:2], h.Size)
	data[3] = 1<<4 | 1<<2
	if h.Tagged {
		data[3] = data[3] | 1<<5
	}
	binary.LittleEndian.PutUint32(data[4:8], h.Source)

	// Frame address.

	binary.LittleEndian.PutUint64(data[8:16], h.Target)
	if h.ResponseRequired {
		data[22] = data[22] | 1<<0
	}
	if h.AcknowledgementRequired {
		data[22] = data[22] | 1<<1
	}
	data[23] = h.Sequence

	// Protocol Header.

	binary.LittleEndian.PutUint16(data[32:34], h.Type)

	return data
}
func (h *header) FromBytes(data []byte) error {
	if len(data) != headerLength {
		return fmt.Errorf("expected %v bytes, got %v", headerLength, len(data))
	}

	h.Size = binary.LittleEndian.Uint16(data[0:2])
	h.Tagged = data[3]&(1<<5) != 0
	h.Source = binary.LittleEndian.Uint32(data[4:8])
	h.Target = binary.LittleEndian.Uint64(data[8:16])
	h.ResponseRequired = data[22]&(1<<1) != 0
	h.AcknowledgementRequired = data[22]&(1<<0) != 0
	h.Sequence = data[23]
	h.Type = binary.LittleEndian.Uint16(data[32:34])

	return nil
}

