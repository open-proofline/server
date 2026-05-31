package main

import (
	"errors"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var errSimulatedNetwork = errors.New("simulated network failure")

type networkProfile struct {
	latency      time.Duration
	jitter       time.Duration
	bandwidth    int64
	offlineEvery int
	offlineFor   time.Duration
	failureRate  float64
	seed         int64
}

func newHTTPClient(cfg config) *http.Client {
	return &http.Client{
		Timeout: cfg.networkTimeout,
		Transport: newNetworkTransport(networkProfile{
			latency:      cfg.networkLatency,
			jitter:       cfg.networkJitter,
			bandwidth:    cfg.networkBandwidth,
			offlineEvery: cfg.networkOfflineEvery,
			offlineFor:   cfg.networkOfflineFor,
			failureRate:  cfg.networkFailureRate,
			seed:         cfg.networkSeed,
		}, http.DefaultTransport),
	}
}

func newNetworkTransport(profile networkProfile, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if !profile.enabled() {
		return base
	}
	return &networkTransport{
		profile: profile,
		base:    base,
		rand:    rand.New(rand.NewSource(profile.seed)),
	}
}

func (p networkProfile) enabled() bool {
	return p.latency > 0 ||
		p.jitter > 0 ||
		p.bandwidth > 0 ||
		p.offlineEvery > 0 ||
		p.offlineFor > 0 ||
		p.failureRate > 0
}

type networkTransport struct {
	profile networkProfile
	base    http.RoundTripper
	mu      sync.Mutex
	rand    *rand.Rand
	count   int
}

func (t *networkTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := t.beforeRequest(); err != nil {
		return nil, err
	}
	if t.profile.bandwidth > 0 && request.Body != nil {
		request = request.Clone(request.Context())
		request.Body = &rateLimitedReadCloser{
			ReadCloser:     request.Body,
			bytesPerSecond: t.profile.bandwidth,
		}
	}
	return t.base.RoundTrip(request)
}

func (t *networkTransport) beforeRequest() error {
	var delay time.Duration
	var fail bool
	var offlineFor time.Duration

	t.mu.Lock()
	t.count++
	count := t.count
	if t.profile.jitter > 0 {
		delay += time.Duration(t.rand.Int63n(int64(t.profile.jitter) + 1))
	}
	if t.profile.failureRate > 0 && t.rand.Float64() < t.profile.failureRate {
		fail = true
	}
	t.mu.Unlock()

	delay += t.profile.latency
	if t.profile.offlineEvery > 0 && count%t.profile.offlineEvery == 0 {
		fail = true
		offlineFor = t.profile.offlineFor
	}
	if delay > 0 {
		time.Sleep(delay)
	}
	if offlineFor > 0 {
		time.Sleep(offlineFor)
	}
	if fail {
		return errSimulatedNetwork
	}
	return nil
}

type rateLimitedReadCloser struct {
	io.ReadCloser
	bytesPerSecond int64
}

func (r *rateLimitedReadCloser) Read(p []byte) (int, error) {
	started := time.Now()
	n, err := r.ReadCloser.Read(p)
	if n > 0 && r.bytesPerSecond > 0 {
		minDuration := time.Duration(int64(time.Second) * int64(n) / r.bytesPerSecond)
		if elapsed := time.Since(started); minDuration > elapsed {
			time.Sleep(minDuration - elapsed)
		}
	}
	return n, err
}
