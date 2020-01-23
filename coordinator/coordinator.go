// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

// Coordinator is the object used to coordinate a cluster of
// load-test agents.
type Coordinator struct {
	config  *CoordinatorConfig
	cluster *cluster.LoadAgentCluster
	monitor *performance.Monitor
}

// Run starts a cluster of load-test agents.
func (c *Coordinator) Run() error {
	mlog.Info("coordinator: ready to drive a cluster of load-test agents", mlog.Int("num_agents", len(c.config.ClusterConfig.Agents)))

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	if err := c.cluster.Run(); err != nil {
		mlog.Error("coordinator: running cluster failed", mlog.Err(err))
		c.cluster.Shutdown()
		return err
	}
	defer c.cluster.Shutdown()

	monitorChan, err := c.monitor.Run()
	if err != nil {
		mlog.Error("coordinator: running monitor failed", mlog.Err(err))
		return err
	}
	defer c.monitor.Stop()

	var lastActionT time.Time
	var lastAlertT time.Time
	var supportedUsers int

	// For now we are keeping all these values constant but in the future they
	// might change based on the state of the feedback loop.
	// Ideally we want the value of users to increment/decrement to react
	// to the speed at which metrics are changing.

	// The value of users to be incremented at each iteration.
	// It should be proportional to the maximum number of users expected to test.
	const incValue = 8
	// The value of users to be decremented at each iteration.
	// It should be proportional to the maximum number of users expected to test.
	const decValue = 8
	// The timespan to wait after a performance degradation alert before
	// incrementing or decrementing users again.
	const restTime = 10 * time.Second

	for {
		var perfStatus performance.Status

		select {
		case <-interruptChannel:
			mlog.Info("coordinator: shutting down")
			return nil
		case perfStatus = <-monitorChan:
		}

		if perfStatus.Alert {
			lastAlertT = time.Now()
		}

		status := c.cluster.Status()
		mlog.Debug("coordinator: cluster status:", mlog.Int("active_users", status.ActiveUsers), mlog.Int64("errors", status.NumErrors))

		// supportedUsers should be estimated in a more clever way in the future.
		// For now we say that the supported number of users is the number of active users that ran
		// for the defined timespan without causing any performance degradation alert.
		if !lastAlertT.IsZero() && !perfStatus.Alert && hasPassed(lastAlertT, restTime) && hasPassed(lastActionT, restTime) {
			supportedUsers = status.ActiveUsers
		}

		mlog.Debug("coordinator: supported users", mlog.Int("supported_users", supportedUsers))

		// We give the feedback loop some rest time in case of performance
		// degradation alerts. We want metrics to stabilize before incrementing/decrementing users again.
		if lastAlertT.IsZero() || lastActionT.IsZero() || hasPassed(lastActionT, restTime) {
			if perfStatus.Alert {
				if err := c.cluster.DecrementUsers(decValue); err != nil {
					mlog.Error("coordinator: failed to decrement users", mlog.Err(err))
				}
				lastActionT = time.Now()
			} else if lastAlertT.IsZero() || hasPassed(lastAlertT, restTime) {
				if status.ActiveUsers < c.config.ClusterConfig.MaxActiveUsers {
					inc := min(incValue, c.config.ClusterConfig.MaxActiveUsers-status.ActiveUsers)
					mlog.Info("coordinator: incrementing active users", mlog.Int("num_users", inc))
					if err := c.cluster.IncrementUsers(inc); err != nil {
						mlog.Error("coordinator: failed to increment users", mlog.Err(err))
					}
					lastActionT = time.Now()
				}
			}
		} else {
			mlog.Debug("coordinator: waiting for metrics to stabilize")
		}
	}
}

// New creates and initializes a new Coordinator for the given config.
// An error is returned if the initialization fails.
func New(config *CoordinatorConfig) (*Coordinator, error) {
	if config == nil {
		return nil, fmt.Errorf("coordinator: config should not be nil")
	}
	if ok, err := config.IsValid(); !ok {
		return nil, err
	}

	cluster, err := cluster.New(config.ClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("coordinator: failed to create cluster: %w", err)
	}

	monitor, err := performance.NewMonitor(config.MonitorConfig)
	if err != nil {
		return nil, fmt.Errorf("coordinator: failed to create performance monitor: %w", err)
	}

	return &Coordinator{
		config:  config,
		cluster: cluster,
		monitor: monitor,
	}, nil
}
