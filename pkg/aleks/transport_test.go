package aleks

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type InterceptingRoundTripper struct {
	Request  *http.Request
	T        *testing.T
	Response *http.Response
}

func (rt *InterceptingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.T.Helper()
	rt.Request = req
	return rt.Response, nil
}

func TestTransport(t *testing.T) {
	trans := aleksTransport()
	assert.True(t, trans.DisableCompression)
	trans.DisableCompression = false
	assert.ObjectsAreEqual(http.DefaultTransport, trans)
}

func TestRoundTripper(t *testing.T) {
	rrt := InterceptingRoundTripper{
		T: t,
		Response: &http.Response{
			Body: ioutil.NopCloser(strings.NewReader("<![CDATA[This is a test]]>")),
		},
	}
	art := &RoundTripper{
		Trans: &rrt,
	}
	url, err := url.Parse("https://example.com/random/path")
	require.NoError(t, err)
	req := http.Request{
		Header: map[string][]string{},
		URL:    url,
	}

	resp, err := art.RoundTrip(&req)
	require.NoError(t, err)

	// Verify that the expected headers are added to the request
	assert.Len(t, req.Header, 3)
	assert.Equal(t, req.Header["Accept"], []string{"*/*"})
	assert.Equal(t, req.Header["User-Agent"], []string{"aleks-client"})
	assert.Equal(t, req.Header["Host"], []string{"example.com"})

	// Verify that the CDATA tags are stripped from the response
	// entity body
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "This is a test", string(respBody))
}
