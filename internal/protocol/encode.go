package protocol

import "fmt"

// Marshal encodes v into its wire format.
// v must implement the Marshaller interface.
func Marshal(v any) ([]byte, error) {
	m, ok := v.(Marshaller)
	if !ok {
		return nil, fmt.Errorf("type %T does not implement Marshaller interface", v)
	}
	return m.MarshalWire()
}
