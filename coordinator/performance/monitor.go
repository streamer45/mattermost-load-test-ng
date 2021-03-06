// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package performance

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

type Monitor struct {
	config     MonitorConfig
	helper     *prometheus.Helper
	stopChan   chan struct{}
	statusChan chan Status
}

// NewMonitor creates and initializes a new Monitor.
func NewMonitor(config MonitorConfig) (*Monitor, error) {
	if err := config.IsValid(); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}
	helper, err := prometheus.NewHelper(config.PrometheusURL)
	if err != nil {
		return nil, fmt.Errorf("performance: failed to create prometheus.Helper: %w", err)
	}
	return &Monitor{
		config:     config,
		helper:     helper,
		stopChan:   make(chan struct{}),
		statusChan: make(chan Status),
	}, nil
}

// Run will start the performance monitoring process.
func (m *Monitor) Run() <-chan Status {
	go func() {
		mlog.Info("monitor: started")
		for {
			m.statusChan <- m.runQueries()
			select {
			case <-m.stopChan:
				mlog.Info("monitor: shutting down")
				return
			case <-time.After(time.Duration(m.config.UpdateIntervalMs) * time.Millisecond):
			}
		}
	}()
	return m.statusChan
}

// Stop will stop the monitoring process.
func (m *Monitor) Stop() {
	mlog.Info("monitor: stop")
	close(m.stopChan)
}

func (m *Monitor) runQueries() Status {
	var status Status
	for _, query := range m.config.Queries {
		select {
		case <-m.stopChan:
			mlog.Info("monitor: exiting query loop")
			return Status{}
		default:
		}
		value, err := m.helper.VectorFirst(query.Query)
		if err != nil {
			mlog.Warn("monitor: error while querying Prometheus:", mlog.String("query_description", query.Description), mlog.Err(err))
			continue
		}
		mlog.Debug("monitor: ran query",
			mlog.String("query_description", query.Description),
			mlog.String("query_returned_value", fmt.Sprintf("%2.8f", value)),
			mlog.String("query_threshold", fmt.Sprintf("%2.8f", query.Threshold)),
		)
		if query.Alert && value >= query.Threshold {
			mlog.Warn("monitor: returned value is above the threshold",
				mlog.String("query_description", query.Description),
				mlog.String("query_returned_value", fmt.Sprintf("%2.8f", value)),
				mlog.String("query_threshold", fmt.Sprintf("%2.8f", query.Threshold)),
			)
			status = Status{Alert: true}
			break
		}
	}
	return status
}
