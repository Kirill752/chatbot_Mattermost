package pool

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	mattermost "github.com/mattermost/mattermost-server/v6/model"
)

type Pool struct {
	ID       int      `json:"id"`
	Title    string   `json:"title"`
	Variants []string `json:"variants"`
}

func (p *Pool) String() string {
	return fmt.Sprintf("id: %d\ntitle: %s\nvariants:%v", p.ID, p.Title, p.Variants)
}

func (p *Pool) MakeResponse(channelId string) *mattermost.Post {
	attachment := &mattermost.SlackAttachment{
		Text:    fmt.Sprintf(`**%s**`, p.Title),
		Actions: make([]*mattermost.PostAction, len(p.Variants)),
	}
	for i, variant := range p.Variants {
		attachment.Actions[i] = &mattermost.PostAction{
			Id:   strconv.Itoa(i),
			Name: variant,
			Type: "button",
		}
	}
	resp := &mattermost.Post{
		ChannelId: channelId,
		Message:   fmt.Sprintf(`**Pool id: %d**`, p.ID),
		Props: map[string]any{
			"attachments": []*mattermost.SlackAttachment{attachment},
		},
	}
	return resp
}

// Format: /опрос [id] "[title]"  variant1, variant2, ...
func Create(msg string) (*Pool, error) {
	newPool := new(Pool)
	rgx := regexp.MustCompile(`^\s*(?i)(?:pool|create\s+pool|опрос|cjplfq|создать\s+опрос)\s+([0-9]+)\s+"([^"]+)"\s+([^,]+(?:,\s*[^,]+)*)$`)
	parts := rgx.FindStringSubmatch(msg)
	if len(parts) < 4 {
		return nil, fmt.Errorf("too few parameters to create a pool")
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("unable to parse pool id: %w", err)
	}
	newPool.ID = id
	newPool.Title = parts[2]
	newPool.Variants = strings.Split(parts[3], " ")
	return newPool, nil
}
