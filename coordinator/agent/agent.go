// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-load-test-ng/api"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

// LoadAgent is the object acting as a client to the load-test agent
// HTTP API.
type LoadAgent struct {
	config Config
	status *loadtest.Status
	client *http.Client
}

var ErrAgentNotFound = errors.New("agent: not found")

// New creates and initializes a new LoadAgent for the given config.
// An error is returned if the initialization fails.
func New(config Config) (*LoadAgent, error) {
	if err := defaults.Validate(config); err != nil {
		return nil, fmt.Errorf("could not validate configartion: %w", err)
	}
	return &LoadAgent{
		config: config,
		status: &loadtest.Status{},
		client: &http.Client{},
	}, nil
}

func (a *LoadAgent) apiRequest(req *http.Request) error {
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == http.StatusNotFound {
			return ErrAgentNotFound
		}
		return fmt.Errorf("agent: bad response status code %d", resp.StatusCode)
	}
	res := &api.Response{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return err
	}
	if res.Error != "" {
		return fmt.Errorf("agent: api request error: %s", res.Error)
	}
	a.status = res.Status
	return nil
}

func (a *LoadAgent) AddUsers(n int) error {
	url := fmt.Sprintf("%s/loadagent/%s/addusers?amount=%d", a.config.ApiURL, a.config.Id, n)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	return a.apiRequest(req)
}

func (a *LoadAgent) RemoveUsers(n int) error {
	url := fmt.Sprintf("%s/loadagent/%s/removeusers?amount=%d", a.config.ApiURL, a.config.Id, n)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	return a.apiRequest(req)
}

func (a *LoadAgent) Start() error {
	a.config.LoadTestConfig.UsersConfiguration.InitialActiveUsers = 0
	var data = struct {
		LoadTestConfig         loadtest.Config
		SimpleControllerConfig *simplecontroller.Config `json:",omitempty"`
		SimulControllerConfig  *simulcontroller.Config  `json:",omitempty"`
	}{
		LoadTestConfig: a.config.LoadTestConfig,
	}

	var err error
	switch a.config.LoadTestConfig.UserControllerConfiguration.Type {
	case loadtest.UserControllerSimple:
		var scc *simplecontroller.Config
		scc, err = simplecontroller.ReadConfig("")
		data.SimpleControllerConfig = scc
	case loadtest.UserControllerSimulative:
		var scc *simulcontroller.Config
		scc, err = simulcontroller.ReadConfig("")
		data.SimulControllerConfig = scc
	}
	if err != nil {
		return err
	}

	configData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// TODO: unify the API requests, making the following code less verbose/repetitive.
	url := fmt.Sprintf("%s/loadagent/create?id=%s", a.config.ApiURL, a.config.Id)
	req, err := http.NewRequest("POST", url, bytes.NewReader(configData))
	if err != nil {
		return err
	}
	if err := a.apiRequest(req); err != nil {
		return err
	}

	url = fmt.Sprintf("%s/loadagent/%s/run", a.config.ApiURL, a.config.Id)
	req, err = http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	if err := a.apiRequest(req); err != nil {
		return err
	}

	mlog.Info("agent: agent created", mlog.String("agent_id", a.config.Id))

	return nil
}

func (a *LoadAgent) Stop() error {
	url := fmt.Sprintf("%s/loadagent/%s", a.config.ApiURL, a.config.Id)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	if err := a.apiRequest(req); err != nil {
		return err
	}

	mlog.Info("agent: agent destroyed", mlog.String("agent_id", a.config.Id))

	return nil
}

func (a *LoadAgent) Status() *loadtest.Status {
	return a.status
}
