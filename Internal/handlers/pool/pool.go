package pool

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"vk_tarantool/Internal/application"

	mattermost "github.com/mattermost/mattermost-server/v6/model"
)

type Request struct {
	ID       int      `json:"id"`
	Title    string   `json:"title"`
	Variants []string `json:"variants"`
}

func New(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "internal.handlers.pool.New"
		log := &app.Logger
		log = log.With(slog.String("op", op),
			slog.String("request_URI", r.RequestURI))
		// парсинг запроса
		decoder := json.NewDecoder(r.Body)
		var req Request
		err := decoder.Decode(&req)
		if err != nil {
			log.Error("unable to parse request", slog.Attr{
				Key:   "error",
				Value: slog.StringValue(err.Error()),
			})
		}
		log.Info("request body decoded", slog.Any("request", req))
		// TODO:  добавить сохранение в базе данных
		// response := mattermost.CommandResponse{
		// 	ResponseType: mattermost.CommandResponseTypeInChannel,
		// 	Text:         "Опрос создан!",
		// 	Attachments: []*mattermost.SlackAttachment{{
		// 		Actions: []*mattermost.PostAction{
		// 			{
		// 				Name: "Да",
		// 				Type: "button",
		// 				Integration: &mattermost.PostActionIntegration{
		// 					URL: "http://ваш-сервер:8080/vote",
		// 				},
		// 			},
		// 		},
		// 	}},
		// }
		// json.NewEncoder(w).Encode(response)
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
	}
}
