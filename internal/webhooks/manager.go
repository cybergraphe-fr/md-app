package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrUnsafeURL = errors.New("webhook URL is not allowed")

// Event types dispatched by the system.
const (
	EventFileCreated = "file.created"
	EventFileUpdated = "file.updated"
	EventFileDeleted = "file.deleted"
)

// Webhook represents a registered webhook endpoint.
type Webhook struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"` // e.g. ["file.created","file.updated"]
	Secret    string    `json:"-"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type webhookPersist struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type WebhookCreated struct {
	Webhook
	Secret string `json:"secret"`
}

// Manager handles webhook storage, lifecycle and async dispatch.
type Manager struct {
	mu         sync.RWMutex
	webhooks   []Webhook
	configPath string
	client     *http.Client
}

// New creates a Manager, loading existing webhooks from configPath.
// configPath is typically {storagePath}/.webhooks.json.
func New(configPath string) *Manager {
	m := &Manager{
		configPath: configPath,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: ssrfSafeDialer,
			},
		},
	}
	if err := m.load(); err != nil {
		slog.Warn("webhooks: could not load config, starting empty",
			"path", configPath,
			"error", err,
		)
	}
	return m
}

// ---- persistence ----

func (m *Manager) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no config yet
		}
		return err
	}
	var persisted []webhookPersist
	if err := json.Unmarshal(data, &persisted); err != nil {
		return fmt.Errorf("parse webhooks config: %w", err)
	}
	m.webhooks = make([]Webhook, len(persisted))
	for i, hook := range persisted {
		m.webhooks[i] = Webhook{
			ID:        hook.ID,
			URL:       hook.URL,
			Events:    hook.Events,
			Secret:    hook.Secret,
			Active:    hook.Active,
			CreatedAt: hook.CreatedAt,
		}
	}
	return nil
}

func (m *Manager) save() error {
	persisted := make([]webhookPersist, len(m.webhooks))
	for i, hook := range m.webhooks {
		persisted[i] = webhookPersist{
			ID:        hook.ID,
			URL:       hook.URL,
			Events:    hook.Events,
			Secret:    hook.Secret,
			Active:    hook.Active,
			CreatedAt: hook.CreatedAt,
		}
	}
	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.configPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, m.configPath)
}

// ---- CRUD ----

// List returns all registered webhooks.
func (m *Manager) List() []Webhook {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Webhook, len(m.webhooks))
	copy(out, m.webhooks)
	return out
}

// Create registers a new webhook endpoint.
func validateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnsafeURL, err)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("%w: only https scheme is allowed", ErrUnsafeURL)
	}
	host := u.Hostname()
	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("%w: DNS resolution failed: %v", ErrUnsafeURL, err)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil && isPrivateIP(ip) {
			return fmt.Errorf("%w: resolved to private IP %s", ErrUnsafeURL, ipStr)
		}
	}
	return nil
}

func isPrivateIP(ip net.IP) bool {
	privateRanges := []*net.IPNet{
		mustParseCIDR("10.0.0.0/8"),
		mustParseCIDR("172.16.0.0/12"),
		mustParseCIDR("192.168.0.0/16"),
		mustParseCIDR("127.0.0.0/8"),
		mustParseCIDR("169.254.0.0/16"),
		mustParseCIDR("::1/128"),
		mustParseCIDR("fc00::/7"),
	}
	for _, network := range privateRanges {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func mustParseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return network
}

func ssrfSafeDialer(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %w", err)
	}
	ips, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("DNS resolution failed: %w", err)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil && isPrivateIP(ip) {
			return nil, fmt.Errorf("connection to private IP %s blocked", ipStr)
		}
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0], port))
}

func (m *Manager) Create(rawURL string, events []string, secret string, active bool) (WebhookCreated, error) {
	if err := validateWebhookURL(rawURL); err != nil {
		return WebhookCreated{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	w := Webhook{
		ID:        uuid.New().String(),
		URL:       rawURL,
		Events:    events,
		Secret:    secret,
		Active:    active,
		CreatedAt: time.Now().UTC(),
	}
	m.webhooks = append(m.webhooks, w)
	if err := m.save(); err != nil {
		// Roll back
		m.webhooks = m.webhooks[:len(m.webhooks)-1]
		return WebhookCreated{}, fmt.Errorf("save config: %w", err)
	}
	return WebhookCreated{Webhook: w, Secret: secret}, nil
}

// Update modifies an existing webhook by ID.
func (m *Manager) Update(id, rawURL string, events []string, secret string, active bool) (Webhook, error) {
	if err := validateWebhookURL(rawURL); err != nil {
		return Webhook{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.webhooks {
		if m.webhooks[i].ID == id {
			m.webhooks[i].URL = rawURL
			m.webhooks[i].Events = events
			m.webhooks[i].Secret = secret
			m.webhooks[i].Active = active
			if err := m.save(); err != nil {
				return Webhook{}, fmt.Errorf("save config: %w", err)
			}
			return m.webhooks[i], nil
		}
	}
	return Webhook{}, fmt.Errorf("webhook %s not found", id)
}

// Delete removes a webhook by ID.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.webhooks {
		if m.webhooks[i].ID == id {
			m.webhooks = append(m.webhooks[:i], m.webhooks[i+1:]...)
			return m.save()
		}
	}
	return fmt.Errorf("webhook %s not found", id)
}

// ---- dispatch ----

// Dispatch sends the event to all active webhooks that subscribe to it.
// Delivery happens asynchronously in goroutines.
func (m *Manager) Dispatch(event string, payload any) {
	m.mu.RLock()
	var targets []Webhook
	for _, w := range m.webhooks {
		if !w.Active {
			continue
		}
		for _, e := range w.Events {
			if e == event {
				targets = append(targets, w)
				break
			}
		}
	}
	m.mu.RUnlock()

	if len(targets) == 0 {
		return
	}

	body, err := json.Marshal(map[string]any{
		"event":     event,
		"payload":   payload,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		slog.Error("webhooks: marshal payload", "error", err)
		return
	}

	for _, w := range targets {
		go m.deliver(w, body)
	}
}

// deliver sends a single webhook request with up to 3 retries and
// exponential backoff (2s, 4s, 8s).
func (m *Manager) deliver(w Webhook, body []byte) {
	const maxRetries = 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			time.Sleep(backoff)
		}

		req, err := http.NewRequest(http.MethodPost, w.URL, bytes.NewReader(body))
		if err != nil {
			slog.Error("webhooks: create request", "url", w.URL, "error", err)
			return // non-retryable
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "MD-Webhook/1.0")

		// HMAC-SHA256 signature
		if w.Secret != "" {
			sig := signPayload(body, w.Secret)
			req.Header.Set("X-Webhook-Signature", "sha256="+sig)
		}

		resp, err := m.client.Do(req)
		if err != nil {
			slog.Warn("webhooks: delivery failed",
				"url", w.URL,
				"attempt", attempt+1,
				"error", err,
			)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			slog.Info("webhooks: delivered",
				"webhook_id", w.ID,
				"url", w.URL,
				"status", resp.StatusCode,
			)
			return
		}

		slog.Warn("webhooks: non-2xx response",
			"url", w.URL,
			"status", resp.StatusCode,
			"attempt", attempt+1,
		)
	}

	slog.Error("webhooks: all retries exhausted",
		"webhook_id", w.ID,
		"url", w.URL,
	)
}

// signPayload computes HMAC-SHA256(body, secret) and returns the hex digest.
func signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
