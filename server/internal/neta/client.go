package neta

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultAppBase    = "https://appapi-pki.chehezhi.cn:18443"
	DefaultEnergyBase = "https://api.chehezhi.cn"
)

type Client struct {
	AppBase    string
	EnergyBase string
	HTTP       *http.Client
}

func NewClient() *Client {
	return &Client{
		AppBase:    DefaultAppBase,
		EnergyBase: DefaultEnergyBase,
		HTTP:       &http.Client{Timeout: 20 * time.Second},
	}
}

const PathRefresh = "/customer/account/info/refreshApiToken"

func (c *Client) Refresh(refreshToken string) (TokenPair, error) {
	form := url.Values{}
	form.Set("refreshToken", refreshToken)
	raw, err := c.postForm(c.AppBase+PathRefresh, "", form)
	if err != nil {
		return TokenPair{}, err
	}
	return DecodeTokenPair(raw)
}

func (c *Client) GetCurrentVehicle(accessToken string) ([]byte, error) {
	return c.postJSON(c.AppBase+"/pivot/mds-api/vehicleAccount/1.0/getCurrentVehicle", accessToken, "{}")
}

func (c *Client) GetAppVehicleData(accessToken, vin string) ([]byte, error) {
	form := url.Values{}
	form.Set("vin", vin)
	form.Set("types", "")
	return c.postForm(c.AppBase+"/pivot/veh-status/vehicle-status-control/1.0/getAppVehicleData", accessToken, form)
}

func (c *Client) QueryEnergyByVin(accessToken, vin string, periodType int) ([]byte, error) {
	form := url.Values{}
	form.Set("vin", vin)
	form.Set("type", fmt.Sprintf("%d", periodType))
	return c.postForm(c.EnergyBase+"/pivot/vehicle-data-api/vehicleEnergyConsumption/1.0/queryEnergyConsumptionByVin", accessToken, form)
}

func (c *Client) postJSON(rawURL, accessToken, body string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	c.auth(req, accessToken)
	return c.do(req)
}

func (c *Client) postForm(rawURL, accessToken string, form url.Values) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	c.auth(req, accessToken)
	return c.do(req)
}

func (c *Client) auth(req *http.Request, accessToken string) {
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	req.Header.Set("Accept", "application/json")
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return b, ErrTokenInvalid
	}
	if resp.StatusCode >= 400 {
		return b, fmt.Errorf("%w: http %d", ErrUpstream, resp.StatusCode)
	}
	return b, nil
}
