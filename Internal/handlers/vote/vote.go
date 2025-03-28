package vote

import (
	"fmt"
	"regexp"
	"strconv"

	mattermost "github.com/mattermost/mattermost-server/v6/model"
)

type Vote struct {
	PoolID         int
	Variant        string
	NumberOfVoters uint
}

func (v *Vote) String() string {
	return fmt.Sprintf("id: %d\ntitle: %s\nNumberOfVoters%v", v.PoolID, v.Variant, v.NumberOfVoters)
}

func (v *Vote) MakeResponse(channelId string) *mattermost.Post {
	// TODO: Запись в базу данных
	resp := &mattermost.Post{
		ChannelId: channelId,
		Message:   fmt.Sprintf("You vote for %s in pool with ID %d.\nThank you!", v.Variant, v.PoolID),
	}
	return resp
}

// Format: vote for variant in [pool_id]
func Create(msg string) (*Vote, error) {
	const op = "Internal.handlers.vote.Create"
	newVote := new(Vote)
	// rgx := regexp.MustCompile(`^\s*(?i)(?:vote\s+for|проголосовать\s+за|голос\s+за)\s+(\w+)\s+([0-9]+)\s*$`)
	rgx := regexp.MustCompile(`(?i)^\s*vote\s+for\s+([^\d\s]+(?:\s+[^\d\s]+)*)\s+(\d+)\s*$`)
	parts := rgx.FindStringSubmatch(msg)
	for _, v := range parts {
		fmt.Println(v)
	}
	if len(parts) < 3 {
		return nil, fmt.Errorf("%s: invalid number of parameters to create a vote", op)
	}
	newVote.Variant = parts[1]
	poolId, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("%s: unable to parse pool id: %w", op, err)
	}
	newVote.PoolID = poolId
	newVote.NumberOfVoters = 0
	return newVote, nil
}
