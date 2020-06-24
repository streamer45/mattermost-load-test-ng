// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

// Coordinator is the object used to coordinate a cluster of
// load-test agents.
type Coordinator struct {
	config  *Config
	cluster *cluster.LoadAgentCluster
	monitor *performance.Monitor
}

// Run starts a cluster of load-test agents.
func (c *Coordinator) Run() error {
	mlog.Info("coordinator: ready to drive a cluster of load-test agents", mlog.Int("num_agents", len(c.config.ClusterConfig.Agents)))

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	defer c.cluster.Shutdown()
	if err := c.cluster.Run(); err != nil {
		mlog.Error("coordinator: running cluster failed", mlog.Err(err))
		return err
	}

	monitorChan := c.monitor.Run()
	defer c.monitor.Stop()

	var lastActionTime, lastAlertTime time.Time

	// For now we are keeping all these values constant but in the future they
	// might change based on the state of the feedback loop.
	// Ideally we want the value of users to increment/decrement to react
	// to the speed at which metrics are changing.

	// The value of users to be incremented at each iteration.
	// TODO: It should be proportional to the maximum number of users expected to test.
	incValue := c.config.NumUsersInc
	// The value of users to be decremented at each iteration.
	// TODO: It should be proportional to the maximum number of users expected to test.
	decValue := c.config.NumUsersDec
	// The timespan to wait after a performance degradation alert before
	// incrementing or decrementing users again.
	restTime := time.Duration(c.config.RestTimeSec) * time.Second

	// TODO: considering making the following values configurable.

	// The threshold at which we consider the load-test done and we are ready to
	// give an answer. The value represent the slope of the best fine line for
	// the gathered samples. This value approaching zero means we have found
	// an equilibrium point.
	stopThreshold := 0.1
	// The timespan to consider when calculating the best fit line. A higher
	// value means considering a higher number of samples which improves the precision of
	// the final result.
	samplesTimeRange := 30 * time.Minute

	var samples []point

	for {
		var perfStatus performance.Status

		select {
		case <-interruptChannel:
			mlog.Info("coordinator: shutting down")
			return nil
		case perfStatus = <-monitorChan:
		}

		if perfStatus.Alert {
			lastAlertTime = time.Now()
		}

		status := c.cluster.Status()
		mlog.Info("coordinator: cluster status:", mlog.Int("active_users", status.ActiveUsers), mlog.Int64("errors", status.NumErrors))

		if !lastAlertTime.IsZero() {
			samples = append(samples, point{
				x: time.Now(),
				y: status.ActiveUsers,
			})
			latest := getLatestSamples(samples, samplesTimeRange)
			if len(latest) > 0 && len(latest) < len(samples) && math.Abs(slope(latest)) < stopThreshold {
				mlog.Info("coordinator done!")
				mlog.Info(fmt.Sprintf("estimated number of supported users is %f", math.Round(avg(latest))))
				return nil
			}
			// We replace older samples which are not needed anymore.
			if len(samples) >= 2*len(latest) {
				copy(samples, latest)
				samples = samples[:len(latest)]
			}
		}

		// We give the feedback loop some rest time in case of performance
		// degradation alerts. We want metrics to stabilize before incrementing/decrementing users again.
		if lastAlertTime.IsZero() || lastActionTime.IsZero() || hasPassed(lastActionTime, restTime) {
			if perfStatus.Alert {
				mlog.Info("coordinator: decrementing active users", mlog.Int("num_users", decValue))
				if err := c.cluster.DecrementUsers(decValue); err != nil {
					mlog.Error("coordinator: failed to decrement users", mlog.Err(err))
				} else {
					lastActionTime = time.Now()
				}
			} else if lastAlertTime.IsZero() || hasPassed(lastAlertTime, restTime) {
				if status.ActiveUsers < c.config.ClusterConfig.MaxActiveUsers {
					inc := min(incValue, c.config.ClusterConfig.MaxActiveUsers-status.ActiveUsers)
					mlog.Info("coordinator: incrementing active users", mlog.Int("num_users", inc))
					if err := c.cluster.IncrementUsers(inc); err != nil {
						mlog.Error("coordinator: failed to increment users", mlog.Err(err))
					} else {
						lastActionTime = time.Now()
					}
				}
			}
		} else {
			mlog.Info("coordinator: waiting for metrics to stabilize")
		}
	}
}

// New creates and initializes a new Coordinator for the given config.
// An error is returned if the initialization fails.
func New(config *Config, ltConfig loadtest.Config) (*Coordinator, error) {
	if config == nil {
		return nil, fmt.Errorf("coordinator: config should not be nil")
	}
	if err := defaults.Validate(config); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}

	cluster, err := cluster.New(config.ClusterConfig, ltConfig)
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
