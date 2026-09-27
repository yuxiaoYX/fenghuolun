package neta

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeLoginToken(t *testing.T) {
	raw := []byte(`{"code":20000,"description":"成功","data":{"type":1,"token":{"access_token":"a","token_type":"bearer","refresh_token":"r","expires_in":604799},"vin":"TESTVIN0000000001"}}`)
	pair, err := DecodeLoginToken(raw)
	if err != nil || pair.AccessToken != "a" || pair.RefreshToken != "r" || pair.ExpiresIn != 604799 {
		t.Fatalf("%+v %v", pair, err)
	}
}

func TestSendLoginCodeSignsForm(t *testing.T) {
	var header http.Header
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		if r.URL.Path != PathSendCode {
			t.Errorf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":20000,"description":"成功","data":true}`))
	}))
	defer srv.Close()
	c := NewClient()
	c.AppBase = srv.URL
	c.HTTP = srv.Client()
	c.AppKey = "test-app-key"
	c.AppSecret = "test-secret"
	if err := c.SendLoginCode("13800138000"); err != nil {
		t.Fatal(err)
	}
	if header.Get("sign") == "" || header.Get("appKey") != "test-app-key" || header.Get("phoneModel") == "" || header.Get("Cookie") != "" || header.Get("Authorization") != "" {
		t.Fatalf("missing signed headers: %#v", header)
	}
	if header.Get("appId") != accountAppID || header.Get("login_channel") != accountLoginCh {
		t.Fatalf("missing evidenced static headers: %#v", header)
	}
	if !strings.Contains(header.Get("User-Agent"), "CHZ/6.4.5") {
		t.Fatalf("ua %q", header.Get("User-Agent"))
	}
	if body != "phone=13800138000" {
		t.Fatalf("body %q", body)
	}
}

func TestSignRequestSendCodeVector(t *testing.T) {
	got := signRequest(
		"POST",
		"/pivot/account/2.0/sendCodeAndSignCheck",
		map[string]string{
			"appId":     accountAppID,
			"appKey":    "test-app-key",
			"nonce":     "4944849486",
			"timestamp": "1788966309835",
		},
		map[string]string{"phone": "13800138000"},
		nil,
		"test-secret",
	)
	const want = "52ca5b1aae8a8789bbbb9448986a198fd7aeb3049c91f29ed6c7cc49bd886639"
	if got != want {
		t.Fatalf("sign=%s want=%s", got, want)
	}
}

func TestAccountHeadersRequireRuntimeKeys(t *testing.T) {
	c := NewClient()
	req, err := http.NewRequest(http.MethodPost, "http://example.test"+PathSendCode, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.accountHeaders(req, map[string]string{"phone": "13800138000"}, nil); !errors.Is(err, ErrSignRequired) {
		t.Fatalf("empty keys: %v", err)
	}
	c.AppSecret = "test-secret"
	if err := c.accountHeaders(req, map[string]string{"phone": "13800138000"}, nil); !errors.Is(err, ErrSignRequired) {
		t.Fatalf("empty appKey: %v", err)
	}
}

func TestSignRequestJSONPrefix(t *testing.T) {
	got := signRequest("POST", "/x",
		map[string]string{"appId": "A", "appKey": "B", "nonce": "1", "timestamp": "2"},
		nil, []byte(`{"phone":"1"}`), "secret")
	if got != signRequest("POST", "/x",
		map[string]string{"appId": "A", "appKey": "B", "nonce": "1", "timestamp": "2"},
		nil, []byte(`{"phone":"1"}`), "secret") {
		t.Fatal(got)
	}
	plain := signRequest("POST", "/x",
		map[string]string{"appId": "A", "appKey": "B", "nonce": "1", "timestamp": "2"},
		map[string]string{"phone": "1"}, nil, "secret")
	if got == plain {
		t.Fatal("json body must change sign")
	}
}

func TestLoginBySMSBody(t *testing.T) {
	var rawBody string
	var header http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		b, _ := io.ReadAll(r.Body)
		rawBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":20000,"description":"成功","data":{"token":{"access_token":"acc","token_type":"bearer","refresh_token":"ref","expires_in":100}}}`))
	}))
	defer srv.Close()
	c := NewClient()
	c.AppBase = srv.URL
	c.HTTP = srv.Client()
	c.AppKey = "test-app-key"
	c.AppSecret = "test-secret"
	pair, err := c.LoginBySMS("13800138000", "123456")
	if err != nil || pair.RefreshToken != "ref" {
		t.Fatal(err, pair)
	}
	for _, part := range []string{`"phone":"13800138000"`, `"code":"123456"`, `"pushToken":""`, `"appVersion":"6.4.5"`, `"deviceType":"1"`, `"seriesNo":"`} {
		if !strings.Contains(rawBody, part) {
			t.Fatalf("body missing %s: %s", part, rawBody)
		}
	}
	if header.Get("sign") == "" || header.Get("timestamp") == "" || header.Get("nonce") == "" {
		t.Fatalf("missing signed JSON headers: %#v", header)
	}
}

func TestBusinessFailureSignAndSMS(t *testing.T) {
	if err := requireBusinessOK([]byte(`{"code":40001,"description":"签名错误","data":null}`)); !errors.Is(err, ErrSignRequired) {
		t.Fatalf("sign %v", err)
	}
	if err := requireBusinessOK([]byte(`{"code":40002,"description":"验证码错误","data":null}`)); !errors.Is(err, ErrSMSRejected) {
		t.Fatalf("sms %v", err)
	}
}
