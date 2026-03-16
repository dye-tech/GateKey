// Package dns provides a lightweight DNS resolver that serves static records from the database.
package dns

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	mdns "github.com/miekg/dns"
	"go.uber.org/zap"
)

// Record represents a DNS record for the resolver
type Record struct {
	Hostname   string
	IPAddress  string
	RecordType string // "A", "AAAA", "CNAME"
	IsWildcard bool
}

// RecordProvider loads DNS records
type RecordProvider func(ctx context.Context) ([]*Record, error)

// ResolverConfig holds DNS resolver configuration
type ResolverConfig struct {
	ListenAddr      string        // e.g., ":5353" or "10.0.0.1:53"
	UpstreamDNS     []string      // Fallback DNS servers for non-matching queries
	RefreshInterval time.Duration // How often to reload records from DB
}

// Resolver is a lightweight DNS server that resolves static records
type Resolver struct {
	config    ResolverConfig
	records   map[string][]*Record // hostname -> records
	wildcards []*Record            // wildcard records (*.example.com)
	mu        sync.RWMutex
	server    *mdns.Server
	provider  RecordProvider
	logger    *zap.Logger
}

// NewResolver creates a new DNS resolver
func NewResolver(config ResolverConfig, provider RecordProvider, logger *zap.Logger) *Resolver {
	return &Resolver{
		config:   config,
		records:  make(map[string][]*Record),
		provider: provider,
		logger:   logger,
	}
}

// Start begins the DNS server and record refresh loop
func (r *Resolver) Start(ctx context.Context) error {
	// Initial load
	if err := r.refreshRecords(ctx); err != nil {
		r.logger.Warn("Failed initial DNS record load", zap.Error(err))
	}

	// Start refresh loop
	go r.refreshLoop(ctx)

	// Create DNS handler
	mux := mdns.NewServeMux()
	mux.HandleFunc(".", r.handleQuery)

	r.server = &mdns.Server{
		Addr:    r.config.ListenAddr,
		Net:     "udp",
		Handler: mux,
	}

	r.logger.Info("DNS resolver started", zap.String("addr", r.config.ListenAddr))

	go func() {
		<-ctx.Done()
		_ = r.server.Shutdown()
	}()

	return r.server.ListenAndServe()
}

func (r *Resolver) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(r.config.RefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.refreshRecords(ctx); err != nil {
				r.logger.Warn("Failed to refresh DNS records", zap.Error(err))
			}
		}
	}
}

func (r *Resolver) refreshRecords(ctx context.Context) error {
	records, err := r.provider(ctx)
	if err != nil {
		return err
	}

	newRecords := make(map[string][]*Record)
	var newWildcards []*Record

	for _, rec := range records {
		key := strings.ToLower(rec.Hostname)
		if rec.IsWildcard {
			newWildcards = append(newWildcards, rec)
		} else {
			newRecords[key] = append(newRecords[key], rec)
		}
	}

	r.mu.Lock()
	r.records = newRecords
	r.wildcards = newWildcards
	r.mu.Unlock()

	r.logger.Debug("DNS records refreshed", zap.Int("count", len(records)))
	return nil
}

func (r *Resolver) handleQuery(w mdns.ResponseWriter, req *mdns.Msg) {
	msg := new(mdns.Msg)
	msg.SetReply(req)
	msg.Authoritative = true

	for _, q := range req.Question {
		name := strings.TrimSuffix(strings.ToLower(q.Name), ".")

		r.mu.RLock()
		// Try exact match first
		records, found := r.records[name]
		if !found {
			// Try wildcard match
			records = r.matchWildcard(name)
			found = len(records) > 0
		}
		r.mu.RUnlock()

		if found {
			for _, rec := range records {
				switch q.Qtype {
				case mdns.TypeA:
					if rec.RecordType == "A" || rec.RecordType == "" {
						ip := net.ParseIP(rec.IPAddress)
						if ip != nil && ip.To4() != nil {
							msg.Answer = append(msg.Answer, &mdns.A{
								Hdr: mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeA, Class: mdns.ClassINET, Ttl: 60},
								A:   ip.To4(),
							})
						}
					}
				case mdns.TypeAAAA:
					if rec.RecordType == "AAAA" {
						ip := net.ParseIP(rec.IPAddress)
						if ip != nil {
							msg.Answer = append(msg.Answer, &mdns.AAAA{
								Hdr:  mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeAAAA, Class: mdns.ClassINET, Ttl: 60},
								AAAA: ip,
							})
						}
					}
				case mdns.TypeCNAME:
					if rec.RecordType == "CNAME" {
						msg.Answer = append(msg.Answer, &mdns.CNAME{
							Hdr:    mdns.RR_Header{Name: q.Name, Rrtype: mdns.TypeCNAME, Class: mdns.ClassINET, Ttl: 60},
							Target: mdns.Fqdn(rec.IPAddress),
						})
					}
				}
			}
		}

		// If no answers found, forward to upstream
		if len(msg.Answer) == 0 && len(r.config.UpstreamDNS) > 0 {
			r.forwardToUpstream(w, req)
			return
		}
	}

	w.WriteMsg(msg)
}

func (r *Resolver) matchWildcard(name string) []*Record {
	var matches []*Record
	for _, wc := range r.wildcards {
		// *.example.com matches foo.example.com
		wcDomain := strings.TrimPrefix(strings.ToLower(wc.Hostname), "*.")
		if strings.HasSuffix(name, "."+wcDomain) || name == wcDomain {
			matches = append(matches, wc)
		}
	}
	return matches
}

func (r *Resolver) forwardToUpstream(w mdns.ResponseWriter, req *mdns.Msg) {
	c := new(mdns.Client)
	c.Timeout = 5 * time.Second

	for _, upstream := range r.config.UpstreamDNS {
		resp, _, err := c.Exchange(req, upstream)
		if err == nil {
			w.WriteMsg(resp)
			return
		}
	}

	// All upstreams failed, return SERVFAIL
	msg := new(mdns.Msg)
	msg.SetRcode(req, mdns.RcodeServerFailure)
	w.WriteMsg(msg)
}

// GetRecordCount returns the number of loaded records
func (r *Resolver) GetRecordCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, recs := range r.records {
		count += len(recs)
	}
	return count + len(r.wildcards)
}
