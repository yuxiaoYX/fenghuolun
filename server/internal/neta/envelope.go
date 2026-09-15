package neta

import (
	"encoding/json"
	"fmt"
)

const businessOK = 20000

type Envelope struct {
	Code        int             `json:"code"`
	Description string          `json:"description"`
	Data        json.RawMessage `json:"data"`
}

func ParseEnvelope(raw []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrDecode, err)
	}
	return env, nil
}

func (e Envelope) OK() bool {
	return e.Code == businessOK
}
