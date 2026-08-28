// Package tl defines the core interfaces and representations for Telegram's
// Type Language (TL) binary serialization system.
package tl

import (
	"github.com/mrabhi2k3/telegofer/tl/decoder"
	"github.com/mrabhi2k3/telegofer/tl/encoder"
)

// Object is implemented by all Telegram TL types, constructors, and RPC methods.
type Object interface {
	// TLID returns the 32-bit CRC constructor ID of this TL object.
	TLID() uint32
	// Encode serializes the object into the provided encoder.
	Encode(e *encoder.Encoder) error
	// Decode deserializes the object from the provided decoder.
	Decode(d *decoder.Decoder) error
}

// Function represents a Telegram RPC method call.
type Function interface {
	Object
}
