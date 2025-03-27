package application

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"vk_tarantool/Internal/config"
	"vk_tarantool/Internal/handlers/pool"

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

func New(configPath string) *Application {
	app := &Application{
		Logger: *slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
	app.Logger.Info("starting application")
	app.Logger.Debug("debug massages enabled")

	app.Config = config.MustLoad(configPath)

	app.Client = mattermost.NewAPIv4Client(app.Config.MM_SERVER)
	app.Client.SetToken(app.Config.MM_TOKEN)
	if user, resp, err := app.Client.GetUser("me", ""); err != nil {
		app.Logger.Error("Could not log in", slog.Attr{
			Key:   "error",
			Value: slog.StringValue(err.Error()),
		})
		os.Exit(1)
	} else {
		app.Logger.Debug("success log in", slog.String("Name:", user.FirstName), slog.String("status code", strconv.Itoa(resp.StatusCode)))
		app.Logger.Info("bot logged in mattermost", slog.String("Name:", user.FirstName))
		app.User = user
	}

	if team, resp, err := app.Client.GetTeamByName(app.Config.MM_TEAM, ""); err != nil {
		app.Logger.Error("Could not find team. Is this bot a member?",
			slog.Attr{
				Key:   "error",
				Value: slog.StringValue(err.Error()),
			},
			slog.String("Bot:", app.User.FirstName), slog.String("Team:", app.Config.MM_TEAM))
		os.Exit(1)
	} else {
		app.Logger.Debug("success find team", slog.String("Team:", team.Name), slog.String("status code", strconv.Itoa(resp.StatusCode)))
		app.Logger.Info("success find team", slog.String("Team:", team.Name))
		app.Team = team
	}

	if ch, resp, err := app.Client.GetChannelByName(app.Config.MM_CHANNEL, app.Team.Id, ""); err != nil {
		app.Logger.Error("Could not find channel in team. Is this bot a member?",
			slog.Attr{
				Key:   "error",
				Value: slog.StringValue(err.Error()),
			},
			slog.String("Bot:", app.User.FirstName), slog.String("Team:", app.Config.MM_TEAM),
			slog.String("Channel:", app.Config.MM_CHANNEL))
		os.Exit(1)
	} else {
		app.Logger.Debug("success find channel", slog.String("Team:", app.Team.Name),
			slog.String("Channel:", ch.Name), slog.String("status code", strconv.Itoa(resp.StatusCode)))
		app.Logger.Info("success find channel", slog.String("Chanel:", ch.Name))
		app.Channel = ch
	}
	return app
}

func (app *Application) ListenToEvents() {
	failCount := 0
	path, err := url.Parse(app.Config.MM_SERVER)
	if err != nil {
		app.Logger.Error("Could not parse url!",
			slog.Attr{
				Key:   "error",
				Value: slog.StringValue(err.Error()),
			})
	}
	for {
		app.WebsocketClient, err = mattermost.NewWebSocketClient4(
			fmt.Sprintf("ws://%s", path.Host+path.Path),
			app.Client.AuthToken,
		)
		if err != nil {
			app.Logger.Error("Mattermost websocket disconnected, retrying")
			failCount += 1
			continue
		}
		app.Logger.Info("Mattermost websocket connected")
		app.sendMsgToChan("Hello, I am a bot!")

		app.WebsocketClient.Listen()
		for event := range app.WebsocketClient.EventChannel {
			if event.GetBroadcast().ChannelId != app.Channel.Id {
				continue
			}
			// Ignore other types of events.
			if event.EventType() != mattermost.WebsocketEventPosted {
				continue
			}
			p := &mattermost.Post{}
			err := json.Unmarshal([]byte(event.GetData()["post"].(string)), &p)
			if err != nil {
				app.Logger.Error("Error while marshaling post", slog.Attr{
					Key:   "error",
					Value: slog.StringValue(err.Error()),
				})
			}
			// Ignore messages sent by this bot itself.
			if p.UserId == app.User.Id {
				continue
			}
			// TODO: добавить обработку сообщения голосования
			if pl, err := pool.Create(p.Message); err == nil {
				resp := pl.MakeResponse(app.Channel.Id)
				_, _, err = app.Client.CreatePost(resp)
				if err != nil {
					app.Logger.Error("Error while sending post", slog.Attr{
						Key:   "error",
						Value: slog.StringValue(err.Error()),
					})
				}
			} else {
				app.sendMsgToChan("I see your event")
			}
			fmt.Printf("Event = %v\n", event.GetData())
		}
	}
}

func (app *Application) SetupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if app.WebsocketClient != nil {
				app.sendMsgToChan("Bye!")
				app.Logger.Info("Closing websocket connection")
				app.WebsocketClient.Close()
			}
			app.Logger.Info("Shutting down")
			os.Exit(0)
		}
	}()
}

func (app *Application) sendMsgToChan(msg string) {
	post := &mattermost.Post{
		ChannelId: app.Channel.Id,
		Message:   msg,
	}
	if _, _, err := app.Client.CreatePost(post); err != nil {
		app.Logger.Error("Could not send massage!",
			slog.Attr{
				Key:   "error",
				Value: slog.StringValue(err.Error()),
			},
			slog.String("Bot:", app.User.FirstName), slog.String("Team:", app.Config.MM_TEAM),
			slog.String("Channel:", app.Config.MM_CHANNEL))
	}
	app.Logger.Debug("Massage was sended", slog.String("Massage:", msg),
		slog.String("Bot:", app.User.FirstName), slog.String("Team:", app.Config.MM_TEAM),
		slog.String("Channel:", app.Config.MM_CHANNEL))
}
