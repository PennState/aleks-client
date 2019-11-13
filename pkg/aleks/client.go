package aleks

import (
	"net/http"
	"net/url"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

const (
	aleksEnvconfigPrefix = "aleks"
	aleksURL             = "https://secure.aleks.com/xmlrpc"
)

type Client struct {
	url      string
	username string
	password string
	trans    http.RoundTripper
}

func NewClient(URL, username, password string) (*Client, error) {
	return newClient(URL, username, password, &RoundTripper{})
}

type clientEnvConfig struct {
	URL      string
	Username string `required:"true"`
	Password string `required:"true"`
}

func NewClientFromEnv() (*Client, error) {
	cfg := clientEnvConfig{}
	err := envconfig.Process(aleksEnvconfigPrefix, &cfg)
	if err != nil {
		return nil, err
	}
	log.Info("Client config: ", cfg)
	return newClient(cfg.URL, cfg.Username, cfg.Password, &RoundTripper{})
}

func newClient(URL, username, password string, trans http.RoundTripper) (*Client, error) {
	if URL == "" {
		URL = aleksURL
	}
	_, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	return &Client{
		url:      URL,
		username: username,
		password: password,
		trans:    trans,
	}, nil
}
