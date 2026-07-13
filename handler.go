package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type tokenCache struct {
	body      []byte
	expiresAt time.Time
}

type server struct {
	ctx    context.Context
	server *http.Server

	cacheMu sync.Mutex
	cache   map[string]*tokenCache
}

func (s *server) handleToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var cookies []*network.CookieParam
	var cacheKey string
	for _, cookie := range r.Cookies() {
		cookies = append(cookies, &network.CookieParam{
			Name:  cookie.Name,
			Value: cookie.Value,
			URL:   spotifyURL,
		})
		cacheKey += cookie.Name + "=" + cookie.Value + ";"
	}

	s.cacheMu.Lock()
	if s.cache == nil {
		s.cache = make(map[string]*tokenCache)
	}
	cached, ok := s.cache[cacheKey]
	s.cacheMu.Unlock()

	if ok && time.Now().Before(cached.expiresAt) {
		slog.InfoContext(ctx, "Returning cached token", slog.Time("expiresAt", cached.expiresAt))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Token-Cached", "true")
		_, _ = w.Write(cached.body)
		return
	}

	slog.InfoContext(ctx, "Fetching fresh token from Spotify")
	body, err := s.getAccessTokenPayload(ctx, cookies)
	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			slog.ErrorContext(ctx, "Failed to get access token payload", slog.Any("err", err))
		}
		http.Error(w, "Failed to get access token payload", http.StatusInternalServerError)
		return
	}

	var parsed struct {
		ExpirationMs int64 `json:"accessTokenExpirationTimestampMs"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.ExpirationMs > 0 {
		expiresAt := time.UnixMilli(parsed.ExpirationMs).Add(-5 * time.Minute)
		s.cacheMu.Lock()
		s.cache[cacheKey] = &tokenCache{body: body, expiresAt: expiresAt}
		s.cacheMu.Unlock()
		slog.InfoContext(ctx, "Token cached", slog.Time("expiresAt", expiresAt))
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Token-Cached", "false")
	if _, err = w.Write(body); err != nil {
		slog.ErrorContext(ctx, "Failed to write response", slog.Any("err", err))
	}
}

func (s *server) getAccessTokenPayload(rCtx context.Context, cookies []*network.CookieParam) ([]byte, error) {
	slog.DebugContext(rCtx, "Getting access token payload", slog.Int("cookieCount", len(cookies)))
	ctx, cancel := chromedp.NewContext(s.ctx)
	defer cancel()

	go func() {
		select {
		case <-rCtx.Done():
			cancel()
		case <-ctx.Done():
		}
	}()

	requestIDChan := make(chan network.RequestID, 1)
	defer close(requestIDChan)

	chromedp.ListenTarget(ctx, func(ev any) {
		switch ev := ev.(type) {
		case *network.EventResponseReceived:
			if !strings.HasPrefix(ev.Response.URL, spotifyTokenURL) {
				return
			}
			requestIDChan <- ev.RequestID
		}
	})

	if err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if len(cookies) == 0 {
				return nil
			}

			if err := network.SetCookies(cookies).Do(ctx); err != nil {
				return fmt.Errorf("failed to set cookies: %w", err)
			}

			return nil
		}),
		chromedp.Navigate(spotifyURL),
	); err != nil {
		return nil, err
	}

	var requestID network.RequestID
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case requestID = <-requestIDChan:
	}

	var body []byte
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		body, err = network.GetResponseBody(requestID).Do(ctx)
		return err
	})); err != nil {
		return nil, err
	}

	return body, nil
}
