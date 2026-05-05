package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/ybeliche/discord/config"
)

func HandleQuestionnaire(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) {
	opts := i.ApplicationCommandData().Options
	eventTitle := opts[0].StringValue()
	pollName := opts[1].StringValue()

	poll, err := cfg.FindPoll(pollName)
	if err != nil {
		respond(s, i, "Unknown poll type.")
		return
	}

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

	content := ""
	if cfg.TeamRoleID != "" {
		content = fmt.Sprintf("<@&%s>", cfg.TeamRoleID)
	}

	_, err = s.ChannelMessageSendComplex(i.ChannelID, &discordgo.MessageSend{
		Content: content,
		Poll: &discordgo.Poll{
			Question:         discordgo.PollMedia{Text: eventTitle},
			AllowMultiselect: poll.AllowMultiselect,
			Duration:         poll.DurationHours,
			Answers:          answers,
		},
	})
	if err != nil {
		log.Printf("Failed to post poll: %v", err)
		respond(s, i, "Failed to post poll.")
		return
	}

	respond(s, i, "Poll posted.")
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
