// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-load-test-ng/api"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func RunServerCmdF(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")

	logger.Init(&logger.Settings{
		EnableConsole: true,
		ConsoleLevel:  "ERROR",
		ConsoleJson:   false,
		EnableFile:    true,
		FileLevel:     "INFO",
		FileJson:      true,
		FileLocation:  "ltagent.log",
	})

	mlog.Info("API server started, listening on", mlog.Int("port", port))
	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), api.SetupAPIRouter(newControllerWrapper))
}

func MakeServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "server",
		Short:        "Start API agent",
		SilenceUsage: true,
		RunE:         RunServerCmdF,
	}
	cmd.PersistentFlags().IntP("port", "p", 4000, "Port to listen on")

	return cmd
}
