package main

import (
	"log/slog"

	mattermost "github.com/mattermost/mattermost-server/v6/model"
)

type application struct {
	logger slog.Logger
	client *mattermost.Client4
	user   *mattermost.User
}
