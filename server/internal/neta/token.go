package neta

import (
	"encoding/json"
	"fmt"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func DecodeTokenPair(raw []byte) (TokenPair, error) {
	env, err := ParseEnvelope(raw)
	if err != nil {
		return TokenPair{}, err
	}
	if env.Code == 41141 {
		return TokenPair{}, ErrTokenInvalid
	}
	if !env.OK() {
		return TokenPair{}, fmt.Errorf("%w: code %d", ErrTokenInvalid, env.Code)
	}
	var pair TokenPair
	if err := json.Unmarshal(env.Data, &pair); err != nil {
		return TokenPair{}, fmt.Errorf("%w: token: %v", ErrDecode, err)
	}
	if pair.AccessToken == "" {
		return TokenPair{}, fmt.Errorf("%w: empty access_token", ErrDecode)
	}
	return pair, nil
}
