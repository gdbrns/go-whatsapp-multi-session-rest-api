package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/env"
	"github.com/gdbrns/go-whatsapp-multi-session-rest-api/pkg/log"
)

type Engine struct {
	store        *Store
	httpClient   *http.Client
	queue        chan *deliveryTask
	workers      int
	retryLimit   int
	maxPerDevice int
	enabled      bool
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

type deliveryTask struct {
	webhook WebhookConfig
	event   WebhookEvent
}

func NewEngine(store *Store) *Engine {
	workers, _ := env.GetEnvInt("WEBHOOK_WORKERS")
	if workers <= 0 {
		workers = 4
	}
	retryLimit, _ := env.GetEnvInt("WEBHOOK_RETRY_LIMIT")
	if retryLimit <= 0 {
		retryLimit = 3
	}
	maxPerDevice, _ := env.GetEnvInt("WEBHOOK_MAX_PER_DEVICE")
	if maxPerDevice <= 0 {
		maxPerDevice = 5
	}
	enabled, err := env.GetEnvBool("WEBHOOKS_ENABLED")
	if err != nil {
		enabled = true
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		store:        store,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		queue:        make(chan *deliveryTask, 1000),
		workers:      workers,
		retryLimit:   retryLimit,
		maxPerDevice: maxPerDevice,
		enabled:      enabled,
		ctx:          ctx,
		cancel:       cancel,
	}

	if enabled {
		for i := 0; i < workers; i++ {
			engine.wg.Add(1)
			go engine.worker()
		}
	}

	return engine
}

func (e *Engine) Store() *Store {
	return e.store
}

func (e *Engine) Shutdown() {
	e.cancel()
	close(e.queue)
	e.wg.Wait()
}

func (e *Engine) Dispatch(ctx context.Context, deviceID string, event WebhookEvent) {
	if !e.enabled {
		return
	}

	webhooks, err := e.store.GetActiveWebhooks(ctx, deviceID)
	if err != nil {
		log.SysErr("wh-fetch", err)
		return
	}

	dispatched := 0
	for _, webhook := range webhooks {
		if e.shouldDispatch(webhook, event.EventType) {
			select {
			case e.queue <- &deliveryTask{webhook: webhook, event: event}:
				dispatched++
			default:
				log.Evt("wh", "queue-full", deviceID, string(event.EventType))
			}
		}
	}

	// Log dispatch with count
	if dispatched > 0 {
		log.WH(string(event.EventType), deviceID, dispatched)
	}
}

func (e *Engine) shouldDispatch(webhook WebhookConfig, eventType EventType) bool {
	if len(webhook.Events) == 0 {
		return true
	}
	for _, evt := range webhook.Events {
		if evt == eventType {
			return true
		}
	}
	return false
}

func (e *Engine) worker() {
	defer e.wg.Done()
	for {
		select {
		case <-e.ctx.Done():
			return
		case task, ok := <-e.queue:
			if !ok {
				return
			}
			e.deliver(task)
		}
	}
}

func (e *Engine) deliver(task *deliveryTask) {
	if err := e.validateURL(task.webhook.URL); err != nil {
		log.WHACK(string(task.event.EventType), task.event.DeviceID, task.webhook.ID, false, 0)
		_ = e.store.LogDelivery(context.Background(), task.webhook.ID, task.event.EventType, DeliveryFailed, 0, err.Error())
		return
	}

	payload, err := json.Marshal(task.event)
	if err != nil {
		log.SysErr("wh-marshal", err)
		return
	}

	signature := e.generateSignature(payload, task.webhook.Secret)

	var lastErr error
	for attempt := 1; attempt <= e.retryLimit; attempt++ {
		req, err := http.NewRequestWithContext(context.Background(), "POST", task.webhook.URL, bytes.NewReader(payload))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Signature", signature)
		req.Header.Set("X-Hub-Signature-256", signature)
		req.Header.Set("X-Webhook-Event", string(task.event.EventType))
		req.Header.Set("User-Agent", "WhatsApp-API-MultiSession/1.0")

		resp, err := e.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < e.retryLimit {
				time.Sleep(time.Duration(attempt*2) * time.Second)
			}
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_ = e.store.LogDelivery(context.Background(), task.webhook.ID, task.event.EventType, DeliverySuccess, attempt, "")
			log.WHACK(string(task.event.EventType), task.event.DeviceID, task.webhook.ID, true, attempt)
			return
		}

		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		if attempt < e.retryLimit {
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}

	errorMsg := ""
	if lastErr != nil {
		errorMsg = lastErr.Error()
	}
	_ = e.store.LogDelivery(context.Background(), task.webhook.ID, task.event.EventType, DeliveryFailed, e.retryLimit, errorMsg)
	log.WHACK(string(task.event.EventType), task.event.DeviceID, task.webhook.ID, false, e.retryLimit)
}

func (e *Engine) generateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (e *Engine) validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	if u.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed")
	}

	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") || strings.HasPrefix(host, "172.") {
		return fmt.Errorf("private/local network URLs are not allowed")
	}

	return nil
}
