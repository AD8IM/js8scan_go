package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"
)

type Scanner struct {
	cfg      *Config
	cl       *Client
	lastAct  atomic.Int64 // unix nano of last RX activity
	curLabel atomic.Value // string
	lastSent map[string]time.Time
}

func NewScanner(cfg *Config, cl *Client) *Scanner {
	s := &Scanner{cfg: cfg, cl: cl, lastSent: map[string]time.Time{}}
	s.curLabel.Store("?")
	return s
}

func (s *Scanner) label() string { return s.curLabel.Load().(string) }

func (s *Scanner) noteActivity() { s.lastAct.Store(time.Now().UnixNano()) }

// HandleEvents prints and tracks everything arriving from the API.
func (s *Scanner) HandleEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-s.cl.Events:
			if !ok {
				log.Println("API connection closed")
				return
			}
			switch m.Type {
			case "RX.DIRECTED":
				s.noteActivity()
				fmt.Printf("[%s] DIRECTED %s>%s snr=%d  %s\n", s.label(),
					m.Str("FROM"), m.Str("TO"), m.Int("SNR"), m.Value)
			case "RX.ACTIVITY":
				s.noteActivity()
				fmt.Printf("[%s] ACTIVITY off=%d snr=%d  %s\n", s.label(),
					m.Int("OFFSET"), m.Int("SNR"), m.Value)
			case "RIG.FREQ":
				log.Printf("rig dial=%d offset=%d", m.Int("DIAL"), m.Int("OFFSET"))
			}
		}
	}
}

// HandleLogs prints lines from ALL.TXT / DIRECTED.TXT.
func (s *Scanner) HandleLogs(ctx context.Context, lines <-chan LogLine) {
	for {
		select {
		case <-ctx.Done():
			return
		case l := <-lines:
			s.noteActivity()
			fmt.Printf("[%s] log:%s  %s\n", s.label(), l.Source, l.Text)
		}
	}
}

// Run cycles through the configured frequencies until ctx is cancelled.
func (s *Scanner) Run(ctx context.Context) {
	for i := 0; ; i = (i + 1) % len(s.cfg.Freqs) {
		f := s.cfg.Freqs[i]
		if !s.dwell(ctx, f) {
			return
		}
	}
}

func (s *Scanner) dwell(ctx context.Context, f Freq) bool {
	log.Printf("QSY -> %s (%.3f kHz)", f.Label, float64(f.Hz)/1000)
	if err := s.cl.SetFreq(f.Hz, s.cfg.Offset); err != nil {
		log.Printf("set freq failed: %v", err)
	}
	s.curLabel.Store(f.Label)
	start := time.Now()
	s.lastAct.Store(0)

	// Give the rig a moment to settle before any transmit.
	if !sleep(ctx, 3*time.Second) {
		return false
	}
	s.maybeSend(f)

	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-t.C:
		}
		el := time.Since(start)
		if el >= s.cfg.MaxDwell {
			return true
		}
		if el >= s.cfg.MinDwell {
			la := s.lastAct.Load()
			if la == 0 || time.Since(time.Unix(0, la)) > s.cfg.QuietTime {
				return true
			}
		}
	}
}

func (s *Scanner) maybeSend(f Freq) {
	if !s.cfg.TXEnabled || !s.cfg.SendOn[strings.ToLower(f.Label)] {
		return
	}
	for _, name := range s.cfg.SendList {
		key := f.Label + "/" + name
		if t, ok := s.lastSent[key]; ok && time.Since(t) < s.cfg.MinInterval {
			continue
		}
		text := s.cfg.Messages[name]
		log.Printf("TX [%s] %s", f.Label, text)
		if err := s.cl.SendMessage(text); err != nil {
			log.Printf("send failed: %v", err)
			continue
		}
		s.lastSent[key] = time.Now()
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
