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

// NewClient returns a new Aleks client given an optional URL, a username
// and a password.
func NewClient(url, username, password string) (*Client, error) {
	return newClient(url, username, password, &RoundTripper{})
}

type clientEnvConfig struct {
	URL      string
	Username string `required:"true"`
	Password string `required:"true"`
}

// NewClientFromEnv returns a new Aleks client from environment variables
// as follows:
//
// - ALEKS_URL      (Optional - see default in constants)
// - ALEKS_USERNAME (Required)
// - ALEKS_PASSWORD (Required)
//
// It is important to note that the individual Aleks XMLRPC calls will
// generally required additional parameters.
func NewClientFromEnv() (*Client, error) {
	cfg := clientEnvConfig{}
	err := envconfig.Process(AleksEnvconfigPrefix, &cfg)
	if err != nil {
		return nil, err
	}
	return newClient(cfg.URL, cfg.Username, cfg.Password, &RoundTripper{})
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
