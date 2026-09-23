package neta

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	PathSendCode = "/pivot/account/2.0/sendCodeAndSignCheck"
	PathSMSLogin = "/pivot/account/2.0/accountSafe/registerOrLoginUncheck"

	// HAR 里一次成功登录带的静态客户端标识。不是签名。
	// sign / appKey / Cookie 每次不同，算法未验证，这里不发送、也不编造。
	accountAppID      = "HOZON-B-xKrgEvMt"
	accountAppVersion = "6.4.5"
	accountChannel    = "iOS"
	accountLoginCh    = "1"
	accountDeviceType = "1"
)

type smsLoginBody struct {
	Phone      string `json:"phone"`
	Code       string `json:"code"`
	PushToken  string `json:"pushToken"`
	SeriesNo   string `json:"seriesNo"`
	AppVersion string `json:"appVersion"`
	DeviceType string `json:"deviceType"`
}

func (c *Client) SendLoginCode(phone string) error {
	form := url.Values{}
	form.Set("phone", phone)
	raw, err := c.postAccountForm(c.AppBase+PathSendCode, form)
	if err != nil {
		return err
	}
	return requireBusinessOK(raw)
}

func (c *Client) LoginBySMS(phone, code string) (TokenPair, error) {
	body, err := json.Marshal(smsLoginBody{
		Phone:      phone,
		Code:       code,
		PushToken:  "",
		SeriesNo:   newSeriesNo(),
		AppVersion: accountAppVersion,
		DeviceType: accountDeviceType,
	})
	if err != nil {
		return TokenPair{}, err
	}
	raw, err := c.postAccountJSON(c.AppBase+PathSMSLogin, body)
	if err != nil {
		return TokenPair{}, err
	}
	return DecodeLoginToken(raw)
}

func DecodeLoginToken(raw []byte) (TokenPair, error) {
	env, err := requireEnvelope(raw)
	if err != nil {
		return TokenPair{}, err
	}
	var wrap struct {
		Token TokenPair `json:"token"`
	}
	if err := json.Unmarshal(env.Data, &wrap); err != nil {
		return TokenPair{}, fmt.Errorf("%w: login token: %v", ErrDecode, err)
	}
	if wrap.Token.AccessToken == "" || wrap.Token.RefreshToken == "" {
		return TokenPair{}, fmt.Errorf("%w: empty login token", ErrDecode)
	}
	return wrap.Token, nil
}

func requireBusinessOK(raw []byte) error {
	_, err := requireEnvelope(raw)
	return err
}

func requireEnvelope(raw []byte) (Envelope, error) {
	env, err := ParseEnvelope(raw)
	if err != nil {
		return Envelope{}, err
	}
	if env.OK() {
		return env, nil
	}
	return Envelope{}, businessFailure(env)
}

type APIError struct {
	Code        int
	Description string
}

func (e *APIError) Error() string {
	if e == nil || e.Description == "" {
		return "neta upstream"
	}
	return e.Description
}

func (e *APIError) Unwrap() error {
	if e == nil {
		return ErrUpstream
	}
	desc := e.Description
	low := strings.ToLower(desc)
	switch {
	case e.Code == 41141:
		return ErrTokenInvalid
	case strings.Contains(desc, "签") || strings.Contains(low, "sign"):
		return ErrSignRequired
	case strings.Contains(desc, "验证码"):
		return ErrSMSRejected
	default:
		return ErrUpstream
	}
}

func businessFailure(env Envelope) error {
	if env.Code == 41141 {
		return ErrTokenInvalid
	}
	return &APIError{Code: env.Code, Description: strings.TrimSpace(env.Description)}
}

func (c *Client) postAccountForm(rawURL string, form url.Values) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	c.accountHeaders(req)
	return c.do(req)
}

func (c *Client) postAccountJSON(rawURL string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	c.accountHeaders(req)
	return c.do(req)
}

func (c *Client) accountHeaders(req *http.Request) {
	// net/http 的 Header.Set 会把 appId 收成 Appid。官方网关按原始头名判断验签，对不上就回「验签信息缺失」。
	setRaw(req.Header, "Accept", "application/json")
	setRaw(req.Header, "appId", accountAppID)
	setRaw(req.Header, "appVersion", accountAppVersion)
	setRaw(req.Header, "channel", accountChannel)
	setRaw(req.Header, "login_channel", accountLoginCh)
	setRaw(req.Header, "timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	setRaw(req.Header, "nonce", newNonce())
	if c.AppKey != "" {
		setRaw(req.Header, "appKey", c.AppKey)
	}
	setRaw(req.Header, "User-Agent", "fenghuolun")
}

func setRaw(h http.Header, key, value string) {
	h[key] = []string{value}
}

func newNonce() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "0000000000"
	}
	n := binary.BigEndian.Uint64(b[:]) % 10000000000
	return fmt.Sprintf("%010d", n)
}

func newSeriesNo() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
