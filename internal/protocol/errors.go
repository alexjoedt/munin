package protocol

import "fmt"

type PeerErr struct {
	PeerID  string
	Message string
	Err     error
}

func (e *PeerErr) Error() string {
	var msg string
	if e.Message != "" {
		msg = e.Message
	}

	if e.Err != nil {
		if msg != "" {
			msg += ": " + e.Err.Error()
		} else {
			msg = e.Err.Error()
		}
	}

	if msg != "" && e.PeerID != "" {
		msg += fmt.Sprintf(" (peerID: %s)", e.PeerID)
	}

	return msg
}

func (e *PeerErr) Unwrap() error {
	return e.Err
}
