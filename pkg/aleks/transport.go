/*
Copyright 2019 The Pennsylvania State University

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aleks

import (
	"io/ioutil"
	"net"
	"net/http"
	"strings"
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
// by the #RoundTrip method.
type RoundTripper struct{}

// RoundTrip implements https://golang.org/pkg/net/http/#RoundTripper.
// The XML-RPC specification requires the User-Agent and Host headers
// which the kolo/xmlrpc library doesn't honor.  Aleks doesn't appear
// to care if they're missing but we're adding them here for completeness.
// More importantly, this intercepter replaces the default transport
// with one that has compression disabled.  On the response side, the
// XML-RPC library doesn't deal with string values wrapped in CDATA
// tags so this RoundTripper also strips those tags from the result.
func (*RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Set the headers required by the specification
	req.Header["Accept"] = []string{"*/*"}
	req.Header["User-Agent"] = []string{"aleks-client"}
	host := req.URL.Hostname()
	req.Header["Host"] = []string{host}

	// Make the call using the customized transport
	resp, err := aleksTransport().RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Strip CDATA tags
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}
	bodyStr := string(body)
	bodyStr = strings.ReplaceAll(bodyStr, "<![CDATA[", "")
	bodyStr = strings.ReplaceAll(bodyStr, "]]>", "")
	resp.Body = ioutil.NopCloser(strings.NewReader(bodyStr))

	return resp, err
}
