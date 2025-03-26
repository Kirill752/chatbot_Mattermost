package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"vk_tarantool/Internal/application"
	"vk_tarantool/Internal/config"
	"vk_tarantool/Internal/handlers/pool"

	mattermost "github.com/mattermost/mattermost-server/v6/model"
)

func main() {
	app := application.New()
	app.Logger.Info("starting bot")
	app.Logger.Debug("debug massages enabled")

	app.Config = config.MustLoad("./Config/conf.yaml")

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

	// Find and save the bot's team to app struct.
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
	setupGracefulShutdown(app)
	registerSlashCommand(app, "pool")
	sendMsgToChan(app, "Hello!")
	http.HandleFunc("/pool/", pool.New(app))
	if err := http.ListenAndServe("localhost:8080", nil); err != nil {
		log.Fatal(err)
	}
	// listenToEvents(app)
}

func listenToEvents(app *application.Application) {
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

		app.WebsocketClient.Listen()
		for event := range app.WebsocketClient.EventChannel {
			if event.GetBroadcast().ChannelId != app.Channel.Id {
				continue
			}
			// Ignore other types of events.
			if event.EventType() != mattermost.WebsocketEventPosted {
				continue
			}
			// Ignore messages sent by this bot itself.
			p := &mattermost.Post{}
			err := json.Unmarshal([]byte(event.GetData()["post"].(string)), &p)
			if err != nil {
				app.Logger.Error("Error while marshaling post", slog.Attr{
					Key:   "error",
					Value: slog.StringValue(err.Error()),
				})
			}
			if p.UserId == app.User.Id {
				continue
			}
			if p.Message == "create pool" {
				response := &mattermost.Post{
					ChannelId: app.Channel.Id,
					Message:   "**Опрос: Какой ваш любимый язык программирования?**",
					Props: map[string]any{
						"attachments": []*mattermost.SlackAttachment{
							{
								Actions: []*mattermost.PostAction{
									{
										Name: "Go",
										Type: "button",
										Integration: &mattermost.PostActionIntegration{
											URL: "http://your-server.com/vote",
											Context: map[string]interface{}{
												"option": "go",
											},
										},
									},
									{
										Name: "Python",
										Type: "button",
										Integration: &mattermost.PostActionIntegration{
											URL: "http://your-server.com/vote",
											Context: map[string]interface{}{
												"option": "python",
											},
										},
									},
								},
							},
						},
					},
				}

				_, _, err = app.Client.CreatePost(response)
				if err != nil {
					app.Logger.Error("Error while sending post", slog.Attr{
						Key:   "error",
						Value: slog.StringValue(err.Error()),
					})
				}
			} else {
				sendMsgToChan(app, "I see your event")
			}
			fmt.Printf("Event = %v\n", event.GetData())
		}
	}
}

func setupGracefulShutdown(app *application.Application) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if app.WebsocketClient != nil {
				sendMsgToChan(app, "Bye!")
				app.Logger.Info("Closing websocket connection")
				app.WebsocketClient.Close()
			}
			app.Logger.Info("Shutting down")
			os.Exit(0)
		}
	}()
}

func sendMsgToChan(app *application.Application, msg string) {
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

func registerSlashCommand(app *application.Application, trigger string) {
	if checkCommandExist(app, trigger) {
		return
	}
	command := &mattermost.Command{
		TeamId:       app.Team.Id,
		Trigger:      "pool",
		Description:  "create a poll",
		URL:          app.Config.MM_SERVER + "/pool",
		Method:       mattermost.CommandMethodPost,
		AutoComplete: true,
	}
	_, resp, err := app.Client.CreateCommand(command)
	if err != nil {
		app.Logger.Error("can not add command")
	}
	app.Logger.Info("command was seccessfully created", slog.String("command", command.Trigger),
		slog.String("status code", strconv.Itoa(resp.StatusCode)))
}

func checkCommandExist(app *application.Application, trigger string) bool {
	commands, resp, err := app.Client.ListCommands(app.Team.Id, false)
	if err != nil {
		app.Logger.Error("error receiving commands", slog.Attr{
			Key:   "error",
			Value: slog.StringValue(err.Error()),
		})
		os.Exit(1)
	}
	for _, cmd := range commands {
		if cmd.Trigger == trigger {
			app.Logger.Info("command exists", slog.String("command", trigger),
				slog.String("status code", strconv.Itoa(resp.StatusCode)))
			return true
		}
	}
	return false
}
