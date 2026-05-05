package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/ybeliche/discord/config"
)

func RegisterCommands(s *discordgo.Session, guildID string, cfg *config.Config) {
	polls := cfg.AllPolls()
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(polls))
	for _, p := range polls {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  p.Label,
			Value: p.Name,
		})
	}

	cmd := &discordgo.ApplicationCommand{
		Name:        "questionnaire",
		Description: "Post an event poll to this channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "event",
				Description: "Event title (e.g. Субботний ивент (02.05) UEC в 22:00 по МСК)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "type",
				Description: "Which poll template to use",
				Required:    true,
				Choices:     choices,
			},
		},
	}

	_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
	if err != nil {
		log.Printf("Failed to register /questionnaire: %v\n", err)
	} else {
		log.Println("Registered /questionnaire")
	}
}
