// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gencontroller

import (
	"errors"
	"math"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
)

// Config holds information about the data to be generated by the
// GenController.
type Config struct {
	// The target number of teams to be created.
	NumTeams int64 `default:"2" validate:"range:[0,]"`
	// The target number of channels to be created.
	NumChannels int64 `default:"20" validate:"range:[0,]"`
	// The target number of posts to be created.
	NumPosts int64 `default:"1000" validate:"range:[0,]"`
	// The target number of reactions to be created.
	NumReactions int64 `default:"200" validate:"range:[0,]"`

	// The percentage of replies to be created.
	PercentReplies float64 `default:"0.5" validate:"range:[0,1]"`

	// Percentages of channels to be created, grouped by type.
	// The total sum of these values must be equal to 1.

	// The percentage of public channels to be created.
	PercentPublicChannels float64 `default:"0.2" validate:"range:[0,1]"`
	// The percentage of private channels to be created.
	PercentPrivateChannels float64 `default:"0.1" validate:"range:[0,1]"`
	// The percentage of direct channels to be created.
	PercentDirectChannels float64 `default:"0.6" validate:"range:[0,1]"`
	// The percentage of group channels to be created.
	PercentGroupChannels float64 `default:"0.1" validate:"range:[0,1]"`
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config

	if err := defaults.ReadFromJSON(configFilePath, "./config/gencontroller.json", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// IsValid reports whether a given gencontroller.Config is valid or not.
// Returns an error if the validation fails.
func (c *Config) IsValid() error {
	percentChannels := c.PercentPublicChannels + c.PercentPrivateChannels + c.PercentDirectChannels + c.PercentGroupChannels
	if (math.Round(percentChannels*100) / 100) != 1 {
		return errors.New("sum of percentages for channels should be equal to 1")
	}

	return nil
}
