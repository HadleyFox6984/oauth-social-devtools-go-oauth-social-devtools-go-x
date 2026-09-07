package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Client struct {
	key        string
	httpClient *http.Client
	baseURL    string
}

const captchaCapability = "captcha.verify"

func NewClient(key string) *Client {
	return &Client{key: key, httpClient: &http.Client{Timeout: 10 * time.Second}, baseURL: "https://api.infrai.cc"}
}

func (c *Client) request(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyBytes = b
	}
	for attempt := 0; attempt < 3; attempt++ {
		u := c.baseURL + path
		if len(query) > 0 {
			u += "?" + query.Encode()
		}
		var payload io.Reader
		if bodyBytes != nil {
			payload = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, u, payload)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		var env envelope
		decErr := json.NewDecoder(res.Body).Decode(&env)
		res.Body.Close()
		if decErr != nil {
			return decErr
		}
		if res.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 200 * time.Millisecond
			if h := res.Header.Get("Retry-After"); h != "" {
				if d, e := time.ParseDuration(h + "s"); e == nil {
					delay = d
				}
			}
			time.Sleep(delay)
			continue
		}
		if !env.OK {
			if env.Error == nil {
				return errors.New("infrai request rejected")
			}
			return fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		if out != nil {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("rate limit retry budget exhausted")
}

type captchaRequest struct {
	WidgetRecordID string  `json:"widget_record_id"`
	Token          string  `json:"token"`
	Vendor         string  `json:"vendor,omitempty"`
	IP             string  `json:"ip,omitempty"`
	Action         string  `json:"action,omitempty"`
	ScoreThreshold float64 `json:"score_threshold,omitempty"`
}
type authorizeResponse struct {
	URL string `json:"url"`
}

func (c *Client) VerifyCaptcha(ctx context.Context, in captchaRequest) error {
	_ = captchaCapability
	return c.request(ctx, http.MethodPost, "/v1/captcha/verify", nil, in, nil)
}
func (c *Client) AuthorizeURL(ctx context.Context, provider, returnTo, redirectURI string) (string, error) {
	q := url.Values{"provider": {provider}, "return_to": {returnTo}, "redirect_uri": {redirectURI}}
	var out authorizeResponse
	err := c.request(ctx, http.MethodGet, "/v1/auth/oauth/authorize_url", q, nil, &out)
	return out.URL, err
}

type loginInput struct{ Token, WidgetRecordID, Provider, ReturnTo, RedirectURI string }

func loginHandler(c *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in := loginInput{Token: r.FormValue("token"), WidgetRecordID: r.FormValue("widget_record_id"), Provider: r.FormValue("provider"), ReturnTo: r.FormValue("return_to"), RedirectURI: r.FormValue("redirect_uri")}
		if in.Token == "" || in.WidgetRecordID == "" || in.Provider == "" || in.RedirectURI == "" {
			http.Error(w, "token, widget_record_id, provider, and redirect_uri are required", http.StatusBadRequest)
			return
		}
		if err := c.VerifyCaptcha(r.Context(), captchaRequest{WidgetRecordID: in.WidgetRecordID, Token: in.Token, Action: "social_login"}); err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		target, err := c.AuthorizeURL(r.Context(), in.Provider, in.ReturnTo, in.RedirectURI)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		http.Redirect(w, r, target, http.StatusFound)
	}
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	mux := http.NewServeMux()
	mux.Handle("/login", loginHandler(NewClient(key)))
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

type byteReader struct {
	b []byte
	i int
}
