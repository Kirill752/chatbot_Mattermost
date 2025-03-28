package application

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	storage "vk_tarantool/Internal/Storage"
	"vk_tarantool/Internal/config"
	"vk_tarantool/Internal/handlers/pool"
	"vk_tarantool/Internal/handlers/vote"

	mattermost "github.com/mattermost/mattermost-server/v6/model"
)

type Application struct {
	Config          *config.Config
	Client          *mattermost.Client4
	WebsocketClient *mattermost.WebSocketClient
	User            *mattermost.User
	Channel         *mattermost.Channel
	Team            *mattermost.Team
	Logger          *slog.Logger
	DB              *storage.Storage
}

func New(configPath string) *Application {
	app := &Application{
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
	db, err := storage.New("127.0.0.1:3301", "storage", "passw0rd")
	if err != nil {
		app.Logger.Error("Could not init database", slog.Attr{
			Key:   "error",
			Value: slog.StringValue(err.Error()),
		})
		os.Exit(1)
	}
	app.DB = db
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
			err = app.HadleWebSocketEvent(event)
			if err != nil {
				app.Logger.Error("Error while procces event", slog.Attr{
					Key:   "error",
					Value: slog.StringValue(err.Error()),
				})
				app.sendMsgToChan("Sorry, somthing went wrong :(")
			}
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
			if app.DB != nil {
				app.Logger.Info("Closing connection with data base")
				app.DB.CloseConnection()
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

func (app *Application) HadleWebSocketEvent(event *mattermost.WebSocketEvent) error {
	const op = "Internal.application.HadleWebSocketEvent"
	if event.GetBroadcast().ChannelId != app.Channel.Id {
		return nil
	}
	// Ignore other types of events.
	if event.EventType() != mattermost.WebsocketEventPosted {
		return nil
	}
	post := &mattermost.Post{}
	err := json.Unmarshal([]byte(event.GetData()["post"].(string)), &post)
	if err != nil {
		app.Logger.Error("Error while marshaling post", slog.Attr{
			Key:   "error",
			Value: slog.StringValue(err.Error()),
		})
		app.sendMsgToChan("Sorry, I can not marshal your post...")
		return fmt.Errorf("%s: %w", op, err)
	}
	// Ignore messages sent by this bot itself.
	if post.UserId == app.User.Id {
		return nil
	}
	switch {
	case strings.HasPrefix(post.Message, "pool"):
		err = app.HandleCreatePool(post)
		if err != nil {
			app.Logger.Error("Error while creating pool", slog.Attr{
				Key:   "error",
				Value: slog.StringValue(err.Error()),
			})
			return err
		}
	case strings.HasPrefix(post.Message, "vote"):
		err = app.HandleVote(post)
		if err != nil {
			app.Logger.Error("Error while proccess vote", slog.Attr{
				Key:   "error",
				Value: slog.StringValue(err.Error()),
			})
			return err
		}
	}
	app.Logger.Info("Event success", slog.Attr{
		Key:   "Event",
		Value: slog.AnyValue(event.GetData()),
	})
	return nil
}

func (app *Application) HandleCreatePool(post *mattermost.Post) error {
	const op = "Internal.application.HandleCreatePool"
	if pl, err := pool.Create(post.Message); err == nil {
		err = app.DB.Save(pl)
		if err != nil {
			return fmt.Errorf("%s: unable to save pool in DB %w", op, err)
		}
		resp := pl.MakeResponse(app.Channel.Id)
		_, _, err = app.Client.CreatePost(resp)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	} else {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (app *Application) HandleVote(post *mattermost.Post) error {
	const op = "Internal.application.HandleVote"
	if vt, err := vote.Create(post.Message); err == nil {
		err = app.DB.AddVote(vt.PoolID, vt.Variant)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
		app.sendMsgToChan(fmt.Sprintf("You vote for %s in pool with ID %d.\nThank you!", vt.Variant, vt.PoolID))
	} else {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
