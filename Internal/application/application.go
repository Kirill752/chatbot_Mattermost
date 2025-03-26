package application

import (
	"log/slog"
	"os"
	"vk_tarantool/Internal/config"

	mattermost "github.com/mattermost/mattermost-server/v6/model"
)

type Application struct {
	Config          *config.Config
	Client          *mattermost.Client4
	WebsocketClient *mattermost.WebSocketClient
	User            *mattermost.User
	Channel         *mattermost.Channel
	Team            *mattermost.Team
	Logger          slog.Logger
}

func New() *Application {
	return &Application{
		Logger: *slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
}
