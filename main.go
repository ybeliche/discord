package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ybeliche/discord/config"
	"github.com/ybeliche/discord/internal/commands"
	"github.com/ybeliche/discord/internal/poll"
	"github.com/ybeliche/discord/internal/scheduler"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN env var not set")
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

	runBotMode(dg, cfg)
}

func runActionMode(s *discordgo.Session, cfg *config.Config) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Printf("Invalid timezone %q, falling back to UTC", cfg.Timezone)
		loc = time.UTC
	}

	today := strings.ToLower(time.Now().In(loc).Weekday().String())

	for id, ch := range cfg.Channels {
		if ch.PickDay != "" && strings.ToLower(ch.PickDay) != today {
			log.Printf("Action: skipping channel %s (pick_day=%s, today=%s)", id, ch.PickDay, today)
			continue
		}
		postedChannels := map[string]bool{}
		for _, sched := range ch.Schedules {
			if sched.ChannelID == "" {
				log.Printf("Action: skipping %q — channel_id not set", sched.Poll)
				continue
			}
			p, err := ch.FindPoll(sched.Poll)
			if err != nil {
				log.Printf("Action: %v", err)
				continue
			}
			next, err := scheduler.NextOccurrence(sched.Day, loc)
			if err != nil {
				log.Printf("Action: skipping %q — %v", sched.Poll, err)
				continue
			}
			title := strings.ReplaceAll(sched.Title, "{date}", next.Format("02.01"))
			log.Printf("Action: posting %q → %s (channel %s)", sched.Poll, title, sched.ChannelID)
			if err := poll.Post(s, sched.ChannelID, title, &p); err != nil {
				log.Printf("Action: failed to post %q: %v", sched.Poll, err)
				continue
			}
			postedChannels[sched.ChannelID] = true
		}

		if ch.TeamRoleID != "" {
			for channelID := range postedChannels {
				msg := fmt.Sprintf("<@&%s>\n\n**Господа! Опросы уже есть, голосуем!**  <:%s:%s>", ch.TeamRoleID, ch.TaggingEmojiName, ch.TaggingEmojiId)
				if _, err := s.ChannelMessageSend(channelID, msg); err != nil {
					log.Printf("Action: failed to tag team in channel %s: %v", channelID, err)
				} else {
					log.Printf("Action: team tagged in channel %s", channelID)
				}
			}
		}
	}

	log.Println("Action: done.")
}

func runBotMode(s *discordgo.Session, cfg *config.Config) {
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

	for _, guildID := range cfg.AllGuildIDs() {
		commands.RegisterCommands(s, guildID, cfg)
	}
	cr := scheduler.Start(s, cfg)
	if cr != nil {
		defer cr.Stop()
	}

	log.Println("Alabama bot is running...")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("Logged in as %s\n", r.User.Username)
}
