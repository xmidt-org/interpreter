package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/xmidt-org/bascule/acquire"
	"github.com/xmidt-org/httpaux"
	"github.com/xmidt-org/httpaux/retry"
	"github.com/xmidt-org/interpreter"
	"go.uber.org/fx"
)

type CodexConfig struct {
	Address       string
	DeviceID      string
	MaxRetryCount int
	JWT           acquire.RemoteBearerTokenAcquirerOptions
	Basic         string
}

type CodexClient struct {
	Address string
	Client  httpaux.Client
	Auth    acquire.Acquirer
}

func Provide() fx.Option {
	return fx.Provide(
		createCodexAuth,
		createClient,
	)
}

func createCodexAuth(config CodexConfig) (acquire.Acquirer, error) {
	defaultAcquirer := &acquire.DefaultAcquirer{}
	jwt := config.JWT
	if jwt.AuthURL != "" && jwt.Buffer > 0 && jwt.Timeout > 0 {
		return acquire.NewRemoteBearerTokenAcquirer(jwt)
	}

	if config.Basic != "" {
		return acquire.NewFixedAuthAcquirer(config.Basic)
	}

	fmt.Fprintln(os.Stderr, "failed to create acquirer")
	return defaultAcquirer, nil
}

func createClient(config CodexConfig, codexAuth acquire.Acquirer) *CodexClient {
	retryConfig := retry.Config{
		Retries:  config.MaxRetryCount,
		Interval: time.Second * 30,
	}

	client := retry.New(retryConfig, new(http.Client))

	return &CodexClient{
		Address: config.Address,
		Auth:    codexAuth,
		Client:  client,
	}
}

func buildGETRequest(address string, auth acquire.Acquirer) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}

	if err := acquire.AddAuth(request, auth); err != nil {
		return nil, err
	}

	return request, nil
}

func (c *CodexClient) sendRequest(req *http.Request) ([]byte, error) {
	resp, err := c.Client.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return body, nil

}

func (c *CodexClient) GetEvents(deviceID string) []interpreter.Event {
	eventList := make([]interpreter.Event, 0)
	request, err := buildGETRequest(fmt.Sprintf("%s/api/v1/device/%s/events", c.Address, deviceID), c.Auth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build request: %v\n", err)
		return eventList
	}

	data, err := c.sendRequest(request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request failed: %v\n", err)
		return eventList
	}

	if err := json.Unmarshal(data, &eventList); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read body: %v\n", err)
		return eventList
	}

	return eventList
}
