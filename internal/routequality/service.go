package routequality

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"
)

const (
	defaultSamples = 7
	connectTimeout = 900 * time.Millisecond
	sampleGap      = 90 * time.Millisecond
	cacheTTL       = 5 * time.Second
)

type Metrics struct {
	TargetID      string    `json:"target_id"`
	MedianRTTMs   *int      `json:"median_rtt_ms"`
	P95RTTMs      *int      `json:"p95_rtt_ms"`
	JitterMs      *int      `json:"jitter_ms"`
	PacketLossPct float64   `json:"packet_loss_pct"`
	Sent          int       `json:"sent"`
	Received      int       `json:"received"`
	MeasuredAt    time.Time `json:"measured_at"`
}

type cachedMeasurement struct {
	metrics Metrics
	until   time.Time
}

type Service struct {
	mu          sync.Mutex
	targets     map[string]Target
	cache       map[string]cachedMeasurement
	targetLocks map[string]*sync.Mutex
	now         func() time.Time
	lookup  func(context.Context, string) ([]net.IP, error)
	dial    func(context.Context, string, string) (net.Conn, error)
}

func New(targets []Target) *Service {
	byID := make(map[string]Target, len(targets))
	for _, target := range targets {
		byID[target.ID] = target
	}

	dialer := &net.Dialer{Timeout: connectTimeout}
	return &Service{
		targets:     byID,
		cache:       make(map[string]cachedMeasurement),
		targetLocks: make(map[string]*sync.Mutex, len(byID)),
		now:         time.Now,
		lookup: func(ctx context.Context, host string) ([]net.IP, error) {
			return net.DefaultResolver.LookupIP(ctx, "ip", host)
		},
		dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, address)
		},
	}
}

func (s *Service) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/route-quality/{target}", func(w http.ResponseWriter, r *http.Request) {
		targetID := r.PathValue("target")
		metrics, err := s.Measure(r.Context(), targetID)
		if errors.Is(err, errUnknownTarget) {
			http.Error(w, "unknown route target", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "route measurement failed", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(metrics)
	})
	return mux
}

var errUnknownTarget = errors.New("unknown route target")

func (s *Service) Measure(ctx context.Context, targetID string) (Metrics, error) {
	s.mu.Lock()
	target, ok := s.targets[targetID]
	if !ok {
		s.mu.Unlock()
		return Metrics{}, errUnknownTarget
	}
	now := s.now().UTC()
	if cached, ok := s.cache[targetID]; ok && now.Before(cached.until) {
		s.mu.Unlock()
		return cached.metrics, nil
	}
	targetLock := s.targetLocks[targetID]
	if targetLock == nil {
		targetLock = &sync.Mutex{}
		s.targetLocks[targetID] = targetLock
	}
	s.mu.Unlock()

	targetLock.Lock()
	defer targetLock.Unlock()

	// Another request may have refreshed this target while we waited.
	s.mu.Lock()
	now = s.now().UTC()
	if cached, ok := s.cache[targetID]; ok && now.Before(cached.until) {
		s.mu.Unlock()
		return cached.metrics, nil
	}
	s.mu.Unlock()

	ips, err := s.lookup(ctx, target.Host)
	if err != nil {
		return Metrics{}, err
	}
	ip, ok := firstPublicIP(ips)
	if !ok {
		return Metrics{}, errors.New("target resolved only to non-public addresses")
	}

	results := make([]*int, 0, defaultSamples)
	for i := 0; i < defaultSamples; i++ {
		started := time.Now()
		probeCtx, cancel := context.WithTimeout(ctx, connectTimeout)
		conn, dialErr := s.dial(probeCtx, "tcp", target.Address(ip))
		cancel()

		if dialErr != nil {
			results = append(results, nil)
		} else {
			_ = conn.Close()
			ms := int(time.Since(started).Milliseconds())
			if ms < 1 {
				ms = 1
			}
			results = append(results, &ms)
		}

		if i != defaultSamples-1 {
			select {
			case <-ctx.Done():
				return Metrics{}, ctx.Err()
			case <-time.After(sampleGap):
			}
		}
	}

	metrics := calculate(target.ID, results, now)
	s.mu.Lock()
	s.cache[targetID] = cachedMeasurement{
		metrics: metrics,
		until:   now.Add(cacheTTL),
	}
	s.mu.Unlock()
	return metrics, nil
}

func firstPublicIP(ips []net.IP) (net.IP, bool) {
	for _, ip := range ips {
		if ip == nil ||
			ip.IsLoopback() ||
			ip.IsPrivate() ||
			ip.IsUnspecified() ||
			ip.IsMulticast() ||
			ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() {
			continue
		}
		return ip, true
	}
	return nil, false
}

func calculate(targetID string, samples []*int, measuredAt time.Time) Metrics {
	receivedOrdered := make([]int, 0, len(samples))
	for _, sample := range samples {
		if sample != nil {
			receivedOrdered = append(receivedOrdered, *sample)
		}
	}
	received := append([]int(nil), receivedOrdered...)
	sort.Ints(received)

	var median *int
	var p95 *int
	if len(received) > 0 {
		medianValue := received[len(received)/2]
		if len(received)%2 == 0 {
			medianValue = int(math.Round(float64(received[len(received)/2-1]+received[len(received)/2]) / 2.0))
		}
		median = &medianValue

		index := int(math.Ceil(float64(len(received))*0.95)) - 1
		if index < 0 {
			index = 0
		}
		if index >= len(received) {
			index = len(received) - 1
		}
		p95Value := received[index]
		p95 = &p95Value
	}

	var jitter *int
	if len(receivedOrdered) >= 2 {
		total := 0
		for i := 1; i < len(receivedOrdered); i++ {
			diff := receivedOrdered[i] - receivedOrdered[i-1]
			if diff < 0 {
				diff = -diff
			}
			total += diff
		}
		value := int(math.Round(float64(total) / float64(len(receivedOrdered)-1)))
		jitter = &value
	}

	loss := 100.0
	if len(samples) > 0 {
		loss = float64(len(samples)-len(received)) / float64(len(samples)) * 100.0
	}

	return Metrics{
		TargetID:      targetID,
		MedianRTTMs:   median,
		P95RTTMs:      p95,
		JitterMs:      jitter,
		PacketLossPct: loss,
		Sent:          len(samples),
		Received:      len(received),
		MeasuredAt:    measuredAt,
	}
}
