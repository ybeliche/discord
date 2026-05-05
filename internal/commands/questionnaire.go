package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/ybeliche/discord/config"
	"github.com/ybeliche/discord/internal/poll"
)

func HandleQuestionnaire(s *discordgo.Session, i *discordgo.InteractionCreate, cfg *config.Config) {
	opts := i.ApplicationCommandData().Options
	eventTitle := opts[0].StringValue()
	pollName := opts[1].StringValue()

	p, err := cfg.FindPoll(pollName)
	if err != nil {
		respond(s, i, "Unknown poll type.")
		return
	}

	if err := poll.Post(s, i.ChannelID, eventTitle, p); err != nil {
		log.Printf("Failed to post poll: %v", err)
		respond(s, i, "Failed to post poll.")
		return
	}

	if cfg.TeamRoleID != "" {
		if _, err := s.ChannelMessageSend(i.ChannelID, fmt.Sprintf("<@&%s>", cfg.TeamRoleID)); err != nil {
			log.Printf("Failed to tag team: %v", err)
		}
	}

	respond(s, i, "Poll posted.")
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.Printf("respond: %v", err)
	}
}
