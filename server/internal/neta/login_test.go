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

func TestSendLoginCodeOmitsSign(t *testing.T) {
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
	if err := c.SendLoginCode("13800138000"); err != nil {
		t.Fatal(err)
	}
	if header.Get("sign") != "" || header.Get("appKey") != "" || header.Get("Cookie") != "" || header.Get("Authorization") != "" {
		t.Fatalf("must not invent sign or copy secrets: %#v", header)
	}
	if header.Get("appId") != accountAppID || header.Get("login_channel") != accountLoginCh {
		t.Fatalf("missing evidenced static headers: %#v", header)
	}
	if body != "phone=13800138000" {
		t.Fatalf("body %q", body)
	}
}

func TestLoginBySMSBody(t *testing.T) {
	var rawBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		rawBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":20000,"description":"成功","data":{"token":{"access_token":"acc","token_type":"bearer","refresh_token":"ref","expires_in":100}}}`))
	}))
	defer srv.Close()
	c := NewClient()
	c.AppBase = srv.URL
	c.HTTP = srv.Client()
	pair, err := c.LoginBySMS("13800138000", "123456")
	if err != nil || pair.RefreshToken != "ref" {
		t.Fatal(err, pair)
	}
	for _, part := range []string{`"phone":"13800138000"`, `"code":"123456"`, `"pushToken":""`, `"appVersion":"6.4.5"`, `"deviceType":"1"`, `"seriesNo":"`} {
		if !strings.Contains(rawBody, part) {
			t.Fatalf("body missing %s: %s", part, rawBody)
		}
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
