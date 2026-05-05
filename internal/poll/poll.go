package poll

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/ybeliche/discord/config"
)

func Post(s *discordgo.Session, channelID, title string, poll *config.Poll, teamRoleID string) error {
	answers := make([]discordgo.PollAnswer, 0, len(poll.Answers))
	for _, a := range poll.Answers {
		media := &discordgo.PollMedia{Text: a.Text}
		switch {
		case a.EmojiID != "":
			media.Emoji = &discordgo.ComponentEmoji{
				Name:     a.EmojiName,
				ID:       a.EmojiID,
				Animated: a.EmojiAnimated,
			}
		case a.Emoji != "":
			media.Emoji = &discordgo.ComponentEmoji{Name: a.Emoji}
		}
		answers = append(answers, discordgo.PollAnswer{Media: media})
	}

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Poll: &discordgo.Poll{
			Question:         discordgo.PollMedia{Text: title},
			AllowMultiselect: poll.AllowMultiselect,
			Duration:         poll.DurationHours,
			Answers:          answers,
		},
	})
	if err != nil {
		return err
	}

	if teamRoleID != "" {
		if _, err := s.ChannelMessageSend(channelID, fmt.Sprintf("<@&%s>", teamRoleID)); err != nil {
			return err
		}
	}
	return nil
}
