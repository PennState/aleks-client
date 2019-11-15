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
	"errors"
	"net/http"
	stdurl "net/url"

	"github.com/kelseyhightower/envconfig"
)

const (
	// AleksEnvconfigPrefix indicates that all environment variables used
	// by this library will start with ALEKS_.
	AleksEnvconfigPrefix = "aleks"

	// AleksDefaultURL is used to access the Aleks service if an alternate
	// URL is not provided via either the parameterized NewClient
	// constructor or the no-parameters NewClientFromEnv constructor.
	AleksDefaultURL = "https://secure.aleks.com/xmlrpc"
)

// Client contains the basic XML-RPC parameters required to make a call
// to the Aleks service.
type Client struct {
	url      string
	username string
	password string
	trans    http.RoundTripper
}

// NewClient returns a new Aleks client given an optional URL and a
// required username and password.
func NewClient(url, username, password string) (*Client, error) {
	rt := RoundTripper{
		Trans: aleksTransport(),
	}
	return newClient(url, username, password, &rt)
}

type clientEnvConfig struct {
	URL      string
	Username string `required:"true"`
	Password string `required:"true"`
}

// NewClientFromEnv returns a new Aleks client from environment variables
// as follows:
//
//   - ALEKS_URL      (Optional - see the default in constants)
//   - ALEKS_USERNAME (Required)
//   - ALEKS_PASSWORD (Required)
//
// It is important to note that the individual Aleks XMLRPC calls will
// generally required additional parameters.
func NewClientFromEnv() (*Client, error) {
	cfg := clientEnvConfig{}
	err := envconfig.Process(AleksEnvconfigPrefix, &cfg)
	if err != nil {
		return nil, err
	}
	rt := RoundTripper{
		Trans: aleksTransport(),
	}
	return newClient(cfg.URL, cfg.Username, cfg.Password, &rt)
}

func newClient(url, username, password string, trans http.RoundTripper) (*Client, error) {
	if url == "" {
		url = AleksDefaultURL
	}
	_, err := stdurl.Parse(url)
	if err != nil {
		return nil, err
	}
	if username == "" || password == "" {
		return nil, errors.New("username and password parameters are both required")
	}
	return &Client{
		url:      url,
		username: username,
		password: password,
		trans:    trans,
	}, nil
}
