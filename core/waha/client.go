package waha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client wraps the WAHA HTTP API. Construct with New.
type Client struct {
	BaseURL string
	APIKey  string
	Session string
	HTTP    *http.Client
}

func New(c Config) *Client {
	return &Client{
		BaseURL: c.BaseURL,
		APIKey:  c.APIKey,
		Session: c.Session,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// do issues a JSON request, attaching X-Api-Key, and decodes into out (if non-nil).
// A non-2xx response is returned as an error so callers can degrade gracefully.
func (c *Client) do(method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, rdr)
	if err != nil {
		return err
	}
	if c.APIKey != "" {
		req.Header.Set("X-Api-Key", c.APIKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("waha %s %s: %d %s", method, path, resp.StatusCode, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (c *Client) GetSession() (*SessionInfo, error) {
	var s SessionInfo
	err := c.do(http.MethodGet, "/api/sessions/"+c.Session, nil, &s)
	return &s, err
}

func (c *Client) sessionAction(action string) (*SessionInfo, error) {
	var s SessionInfo
	err := c.do(http.MethodPost, "/api/sessions/"+c.Session+"/"+action, nil, &s)
	return &s, err
}

func (c *Client) StartSession() (*SessionInfo, error)  { return c.sessionAction("start") }
func (c *Client) StopSession() (*SessionInfo, error)    { return c.sessionAction("stop") }
func (c *Client) RestartSession() (*SessionInfo, error) { return c.sessionAction("restart") }
func (c *Client) LogoutSession() (*SessionInfo, error)  { return c.sessionAction("logout") }

func (c *Client) GetMe() (*MeInfo, error) {
	var m MeInfo
	err := c.do(http.MethodGet, "/api/sessions/"+c.Session+"/me", nil, &m)
	return &m, err
}

// GetQR returns the QR image bytes and its content-type for the pairing screen.
func (c *Client) GetQR() ([]byte, string, error) {
	u := fmt.Sprintf("%s/api/%s/auth/qr?format=image", c.BaseURL, url.PathEscape(c.Session))
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	if c.APIKey != "" {
		req.Header.Set("X-Api-Key", c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("waha qr: %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/png"
	}
	return b, ct, err
}

func (c *Client) SendText(chatID, text string) (*SendResult, error) {
	var res SendResult
	err := c.do(http.MethodPost, "/api/sendText",
		SendTextReq{ChatID: chatID, Text: text, Session: c.Session}, &res)
	return &res, err
}
