package main

import (
	"log"

	mattermost "github.com/mattermost/mattermost-server/v6/model"
)

// MM_TEAM="test"
// MM_TOKEN="ritzg6x5f38c8jkmbzxsjjy81o"
// MM_CHANNEL="town-square"
// MM_SERVER="http://localhost:8065"
// MM_USERNAME="test-bot"

func main() {
	// Новый клиент Mattermost
	url := `http://localhost:8065`
	token := `ritzg6x5f38c8jkmbzxsjjy81o`
	team_name := "newTeam"
	client := mattermost.NewAPIv4Client(url)
	client.SetToken(token)
	var user *mattermost.User
	if u, _, err := client.GetUser("me", ""); err != nil {
		// app.logger.Fatal().Err(err).Msg("Could not log in")
		log.Println("Could not log in")
		log.Fatal(err)
	} else {
		user = u
		log.Printf("user %s logged in mattermost", user.FirstName)
	}

	// Find and save the bot's team to app struct.
	var team *mattermost.Team
	if t, _, err := client.GetTeamByName(team_name, ""); err != nil {
		log.Printf("Could not find %s team. Is this bot a member ?", team_name)
		log.Fatal(err)
	} else {
		team = t
		log.Printf("Team %s was founded\n", team.Name)
	}

	// teams, _, err := client.GetAllTeams("", 0, 100) // 100 - максимальное количество
	// if err != nil {
	// 	log.Fatalf("Ошибка при получении списка команд: %v", err)
	// }

	// // Вывод информации о командах
	// fmt.Println("Доступные команды (teams):")
	// for _, team := range teams {
	// 	fmt.Printf("ID: %s, Имя: %s, Отображаемое имя: %s\n",
	// 		team.Id,
	// 		team.Name,
	// 		team.DisplayName)
	// }

	var channel *mattermost.Channel
	channelName := "town-square"
	if c, _, err := client.GetChannelByName(channelName, team.Id, ""); err != nil {
		log.Printf("Could not find %s channel in team %s. Is this bot a member ?", channelName, team_name)
		log.Fatal(err)
	} else {
		channel = c
		log.Printf("Channel %s was founded\n", channel.Name)
	}

	post := &mattermost.Post{
		ChannelId: channel.Id,
		Message:   "How are you?",
	}
	_, _, err := client.CreatePost(post)
	if err != nil {
		log.Println(err)
	}
}

// func sendMsgToTalkingChannel(app *application, msg string, replyToId string) {
// 	// Note that replyToId should be empty for a new post.
// 	// All replies in a thread should reply to root.

// 	post := &model.Post{}
// 	post.ChannelId = app.mattermostChannel.Id
// 	post.Message = msg

// 	post.RootId = replyToId

// 	if _, _, err := app.mattermostClient.CreatePost(post); err != nil {
// 		app.logger.Error().Err(err).Str("RootID", replyToId).Msg("Failed to create post")
// 	}
// }
