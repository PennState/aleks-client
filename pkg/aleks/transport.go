package aleks

import (
	"net"
	"net/http"
	"time"
)

// aleksTransport is equivalent to the http.DefaultTransport with
// compression disabled.  See https://golang.org/pkg/net/http/#RoundTripper.
func aleksTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}
}

// RoundTripper intercepts HTTP calls and alters the request as described
// below.
type RoundTripper struct{}

// RoundTrip implements https://golang.org/pkg/net/http/#RoundTripper.
// The XML-RPC specification requires the User-Agent and Host headers
// which the kolo/xmlrpc library doesn't honor.  Aleks doesn't appear
// to care if they're missing but we're adding them here for completeness.
// More importantly, this intercepter replaces the default transport
// with one that has compression disabled.
func (art *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header["Accept"] = []string{"*/*"}
	req.Header["User-Agent"] = []string{"aleks-client"}
	host := req.URL.Hostname()
	req.Header["Host"] = []string{host}
	return aleksTransport().RoundTrip(req)
}
