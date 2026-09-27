package neta

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	PathSendCode = "/pivot/account/2.0/sendCodeAndSignCheck"
	PathSMSLogin = "/pivot/account/2.0/accountSafe/registerOrLoginUncheck"

	// HAR 里一次成功登录带的静态客户端标识。appKey / APP_SECRET 只走运行时注入。
	accountAppID      = "HOZON-B-xKrgEvMt"
	accountAppVersion = "6.4.5"
	accountChannel    = "iOS"
	accountLoginCh    = "1"
	accountDeviceType = "1"
	accountPhoneModel = "iPhone X (CDMA)"
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
	params := make(map[string]string, len(form))
	for k, values := range form {
		if len(values) > 0 {
			params[k] = values[0]
		}
	}
	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	if err := c.accountHeaders(req, params, nil); err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) postAccountJSON(rawURL string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	if err := c.accountHeaders(req, nil, body); err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) accountHeaders(req *http.Request, form map[string]string, jsonBody []byte) error {
	appKey := strings.TrimSpace(c.AppKey)
	if appKey == "" || strings.TrimSpace(c.AppSecret) == "" {
		return ErrSignRequired
	}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	nonce := newNonce()
	headers := map[string]string{
		"appId":     accountAppID,
		"appKey":    appKey,
		"nonce":     nonce,
		"timestamp": timestamp,
	}
	params := rawQueryMap(req.URL)
	for k, v := range form {
		params[k] = v
	}
	sign := signRequest(req.Method, req.URL.Path, headers, params, jsonBody, c.AppSecret)
	// net/http 的 Header.Set 会把 appId 收成 Appid。官方网关按原始头名判断验签，对不上就回「验签信息缺失」。
	setRaw(req.Header, "Accept", "*/*")
	setRaw(req.Header, "appId", accountAppID)
	setRaw(req.Header, "appVersion", accountAppVersion)
	setRaw(req.Header, "channel", accountChannel)
	setRaw(req.Header, "login_channel", accountLoginCh)
	setRaw(req.Header, "timestamp", timestamp)
	setRaw(req.Header, "nonce", nonce)
	setRaw(req.Header, "sign", sign)
	setRaw(req.Header, "phoneModel", accountPhoneModel)
	setRaw(req.Header, "appKey", appKey)
	setRaw(req.Header, "User-Agent", "CHZ/6.4.5 (com.hozon.sales.app; build:4; iOS 16.7.11) Alamofire/5.4.4")
	return nil
}

// signRequest 对齐 IPA HZRequestInterceptor（CryptoSwift SHA2.sha256 + toHexString）。
// HAR #58 sendCode 已绿：METHOD+PATH+appid:…appkey:…nonce:…timestamp:…+k:v 表单 + SECRET，
// 再按 urlQueryAllowed 去掉 ":#[]@?/!$&'()+,;=~" 做百分号编码。
func signRequest(method, path string, headers, params map[string]string, jsonBody []byte, secret string) string {
	extra := colonJoin(params)
	if len(jsonBody) > 0 {
		extra += "json:" + string(jsonBody)
	}
	raw := strings.ToUpper(method) + path + colonJoin(headers) + extra + secret
	raw = strings.ReplaceAll(strings.ReplaceAll(raw, " ", ""), "\n", "")
	enc := strings.ReplaceAll(percentEncodeSign(raw), "%20", "")
	sum := sha256.Sum256([]byte(enc))
	return hex.EncodeToString(sum[:])
}

func colonJoin(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if strings.EqualFold(k, "sign") || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(strings.ToLower(k))
		b.WriteByte(':')
		b.WriteString(m[k])
	}
	return b.String()
}

func percentEncodeSign(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_' || c == '*' {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteString(strings.ToUpper(hex.EncodeToString([]byte{c})))
		}
	}
	return b.String()
}

func rawQueryMap(u *url.URL) map[string]string {
	out := map[string]string{}
	if u == nil || u.RawQuery == "" {
		return out
	}
	for _, part := range strings.Split(u.RawQuery, "&") {
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if k == "" {
			continue
		}
		if !ok {
			v = ""
		}
		out[k] = v
	}
	return out
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
