// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gencontroller

import (
	"errors"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/config"

	"github.com/spf13/viper"
)

// Config holds information about the data to be generated by the
// GenController.
type Config struct {
	// The target number of channels to be created.
	NumChannels int
	// The target number of posts to be created.
	NumPosts int
	// The target number of reactions to be created.
	NumReactions int

	// The precentage of replies to be created.
	PercentReplies float64
	// The pecentage of public channels to be created.
	PercentPublicChannels float64
	// The pecentage of private channels to be created.
	PercentPrivateChannels float64
	// The pecentage of direct channels to be created.
	PercentDirectChannels float64
	// The pecentage of group channels to be created.
	PercentGroupChannels float64
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will search a config file in predefined folders.
func ReadConfig(configFilePath string) (*Config, error) {
	v := viper.New()

	configName := "gencontroller"
	v.SetConfigName(configName)
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	v.AddConfigPath("./../config/")
	v.AddConfigPath("./../../../config/")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configFilePath != "" {
		v.SetConfigFile(configFilePath)
	}

	if err := config.ReadConfigFile(v, configName); err != nil {
		return nil, err
	}

	var cfg *Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// IsValid reports whether a given gencontroller.Config is valid or not.
// Returns an error if the validation fails.
func (c *Config) IsValid() error {
	if c.NumChannels < 0 {
		return errors.New("NumChannels should be > 0")
	}

	if c.NumPosts < 0 {
		return errors.New("NumPosts should be > 0")
	}

	if c.NumReactions < 0 {
		return errors.New("NumReactions should be > 0")
	}

	if c.PercentReplies < 0 || c.PercentReplies > 1.0 {
		return errors.New("PercentReplies should be >= 0 and <= 1.0")
	}

	if c.PercentPublicChannels < 0 || c.PercentPublicChannels > 1.0 {
		return errors.New("PercentPublicChannels should be >= 0 and <= 1.0")
	}

	if c.PercentPrivateChannels < 0 || c.PercentPrivateChannels > 1.0 {
		return errors.New("PercentPrivateChannels should be >= 0 and <= 1.0")
	}

	if c.PercentDirectChannels < 0 || c.PercentDirectChannels > 1.0 {
		return errors.New("PercentDirectChannels should be >= 0 and <= 1.0")
	}

	if c.PercentGroupChannels < 0 || c.PercentGroupChannels > 1.0 {
		return errors.New("PercentGroupChannels should be >= 0 and <= 1.0")
	}

	percentChannels := c.PercentPublicChannels + c.PercentPrivateChannels + c.PercentDirectChannels + c.PercentGroupChannels
	if percentChannels != 1 {
		return errors.New("sum of percentages for channels should be equal to 1")
	}

	return nil
}
