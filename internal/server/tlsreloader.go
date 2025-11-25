package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type TLSReloader struct {
	certPath     string
	keyPath      string
	pollInterval time.Duration

	cert        *tls.Certificate
	lastCertMod time.Time
	lastKeyMod  time.Time

	mu     sync.RWMutex
	doneCh chan struct{}
	stopCh chan struct{}
}

func NewTLSReloader(certPath, keyPath string, timer time.Duration) (*TLSReloader, error) {
	absCert, err := filepath.Abs(certPath)
	if err != nil {
		return nil, fmt.Errorf("invalid cert path: %w", err)
	}
	absKey, err := filepath.Abs(keyPath)
	if err != nil {
		return nil, fmt.Errorf("invalid key path: %w", err)
	}

	reloader := &TLSReloader{
		certPath:     absCert,
		keyPath:      absKey,
		pollInterval: timer,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}

	if err := reloader.Reload(true); err != nil {
		return nil, err
	}

	return reloader, nil
}

func (r *TLSReloader) Reload(force bool) error {

	certStat, err := os.Stat(r.certPath)
	if err != nil {
		return fmt.Errorf("failed to stat cert file: %w", err)
	}
	keyStat, err := os.Stat(r.keyPath)
	if err != nil {
		return fmt.Errorf("failed to stat key file: %w", err)
	}

	if !force && certStat.ModTime().Equal(r.lastCertMod) && keyStat.ModTime().Equal(r.lastKeyMod) {
		return nil
	}

	newCert, err := tls.LoadX509KeyPair(r.certPath, r.keyPath)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.cert = &newCert
	r.lastCertMod = certStat.ModTime()
	r.lastKeyMod = certStat.ModTime()
	r.mu.Unlock()

	log.Trace().Msg("reloaded certificate files")
	return nil
}

func (r *TLSReloader) Start() error {
	r.run()
	return nil
}

func (r *TLSReloader) run() {
	defer close(r.doneCh)

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	log.Debug().Str("cert", r.certPath).Dur("poll_interval", r.pollInterval).Msg("starting tls poller")

	for {
		select {
		case <-r.stopCh:
			// Shutdown signal received
			return
		case <-ticker.C:
			if err := r.Reload(false); err != nil {
				log.Err(err).Msg("failed to reload certificate")
			}
		}
	}

}

func (r *TLSReloader) Shutdown(ctx context.Context) error {
	close(r.stopCh)

	select {
	case <-r.doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *TLSReloader) String() string {
	return fmt.Sprintf("TLS-Reloader(%s)", r.certPath)
}

func (r *TLSReloader) GetCertificateFunc(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cert, nil
}
