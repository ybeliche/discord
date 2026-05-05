package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ybeliche/discord/commands"
	"github.com/ybeliche/discord/config"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN env var not set")
	}
	guildID := os.Getenv("GUILD_ID")
	if guildID == "" {
		log.Fatal("GUILD_ID env var not set")
	}

	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages

	if err = dg.Open(); err != nil {
		log.Fatal(err)
	}
	defer dg.Close()

	if os.Getenv("GITHUB_ACTIONS") == "true" {
		runActionMode(dg, cfg)
		return
	}

	runBotMode(dg, cfg, guildID)
}

// runActionMode posts all configured polls with auto-calculated upcoming dates, then exits.
func runActionMode(s *discordgo.Session, cfg *config.Config) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Printf("Invalid timezone %q, falling back to UTC", cfg.Timezone)
		loc = time.UTC
	}

	for _, sched := range cfg.Schedules {
		if sched.ChannelID == "" {
			log.Printf("Action: skipping %q — channel_id not set", sched.Poll)
			continue
		}
		poll, err := cfg.FindPoll(sched.Poll)
		if err != nil {
			log.Printf("Action: %v", err)
			continue
		}
		next, err := commands.NextOccurrence(sched.Day, loc)
		if err != nil {
			log.Printf("Action: skipping %q — %v", sched.Poll, err)
			continue
		}
		title := strings.ReplaceAll(sched.Title, "{date}", next.Format("02.01"))
		log.Printf("Action: posting %q → %s (channel %s)", sched.Poll, title, sched.ChannelID)
		if err := commands.PostPoll(s, sched.ChannelID, title, poll, cfg.TeamRoleID); err != nil {
			log.Printf("Action: failed to post %q: %v", sched.Poll, err)
		}
	}
	fmt.Println("Action: done.")
}

// runBotMode runs the persistent bot with slash commands and cron scheduler.
func runBotMode(s *discordgo.Session, cfg *config.Config, guildID string) {
	s.AddHandler(onReady)
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		switch i.ApplicationCommandData().Name {
		case "questionnaire":
			commands.HandleQuestionnaire(s, i, cfg)
		}
	})

	commands.RegisterCommands(s, guildID, cfg)
	commands.StartScheduler(s, cfg)

	fmt.Println("Alabama bot is running...")
	select {}
}

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Printf("Logged in as %s\n", r.User.Username)
}
