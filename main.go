package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/cobra"
	"github.com/ybeliche/discord/config"
	"github.com/ybeliche/discord/internal/commands"
	"github.com/ybeliche/discord/internal/poll"
	"github.com/ybeliche/discord/internal/scheduler"
)

var DayOfWeek = map[string]string{
	"SUNDAY":    "Воскресенье",
	"MONDAY":    "Понедельник",
	"TUESDAY":   "Вторник",
	"WEDNESDAY": "Среда",
	"THURSDAY":  "Четверг",
	"FRIDAY":    "Пятница",
	"SATURDAY":  "Суббота",
}

func newSession() *discordgo.Session {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN env var not set")
	}
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages
	if err = dg.Open(); err != nil {
		log.Fatal(err)
	}
	return dg
}

func main() {
	var channelID, messageID string

	rootCmd := &cobra.Command{
		Use:   "alabama",
		Short: "Alabama Discord bot",
		RunE: func(cmd *cobra.Command, args []string) error {
			dg := newSession()
			defer dg.Close()

			cfg, err := config.Load("config/config.yaml")
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if os.Getenv("GITHUB_ACTIONS") == "true" {
				runActionMode(dg, cfg)
				return nil
			}

			runBotMode(dg, cfg)
			return nil
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a Discord message by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if channelID == "" || messageID == "" {
				return fmt.Errorf("--channel and --message are required")
			}
			dg := newSession()
			defer dg.Close()
			return runDeleteMode(dg, channelID, messageID)
		},
	}

	deleteCmd.Flags().StringVar(&channelID, "channel", "", "Discord channel ID")
	deleteCmd.Flags().StringVar(&messageID, "message", "", "Discord message ID to delete")

	rootCmd.AddCommand(deleteCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runActionMode(s *discordgo.Session, cfg *config.Config) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Printf("Invalid timezone %q, falling back to UTC", cfg.Timezone)
		loc = time.UTC
	}

	today := strings.ToLower(time.Now().In(loc).Weekday().String())

	type postResult struct {
		poll      string
		channelID string
		err       error
	}

	for id, ch := range cfg.Channels {
		if ch.PickDay != "" && strings.ToLower(ch.PickDay) != today {
			log.Printf("Action: skipping channel %s (pick_day=%s, today=%s)", id, ch.PickDay, today)
			continue
		}

		// When main_schedule is enabled, the channel ignores its own schedules
		// and posts the global cfg.Schedules to its main_schedule.channel_id.
		iterSchedules := ch.Schedules
		if ch.MainSchedule.Enabled {
			log.Printf("Action: main schedule enabled for channel %s → posting to %s", id, ch.MainSchedule.ChannelID)
			iterSchedules = make([]config.Schedule, 0, len(cfg.Schedules))
			for _, rs := range cfg.Schedules {
				iterSchedules = append(iterSchedules, config.Schedule{
					Poll:      rs.Poll,
					ChannelID: ch.MainSchedule.ChannelID,
					Title:     rs.Title,
					Day:       rs.Day,
					At:        rs.At,
				})
			}
		}

		var wg sync.WaitGroup
		results := make(chan postResult, len(iterSchedules))

		for _, sched := range iterSchedules {
			if sched.ChannelID == "" && ch.MainSchedule.ChannelID == "" {
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
			title = strings.ReplaceAll(title, "{day}", DayOfWeek[strings.ToUpper(sched.Day)])
			log.Printf("Action: posting %q → %s (channel %s)", sched.Poll, title, sched.ChannelID)

			wg.Add(1)
			go func(sched config.Schedule, p config.Poll, title string) {
				defer wg.Done()
				results <- postResult{sched.Poll, sched.ChannelID,
					poll.Post(s, sched.ChannelID, title, &p)}
			}(sched, p, title)
		}

		wg.Wait()
		close(results)

		postedChannels := map[string]bool{}
		for r := range results {
			if r.err != nil {
				log.Printf("Action: failed to post %q: %v", r.poll, r.err)
			} else {
				log.Printf("Action: posted %q successfully", r.poll)
				postedChannels[r.channelID] = true
			}
		}

		if ch.TeamRoleID != "" {
			var tagWg sync.WaitGroup
			msg := fmt.Sprintf("<@&%s>\n\n**Господа! Опросы уже есть, голосуем!**", ch.TeamRoleID)
			if ch.TaggingEmojiName != "" && ch.TaggingEmojiId != "" {
				msg += fmt.Sprintf("  <:%s:%s>", ch.TaggingEmojiName, ch.TaggingEmojiId)
			}
			for channelID := range postedChannels {
				tagWg.Add(1)
				go func(channelID string) {
					defer tagWg.Done()
					if _, err := s.ChannelMessageSend(channelID, msg); err != nil {
						log.Printf("Action: failed to tag team in channel %s: %v", channelID, err)
					} else {
						log.Printf("Action: team tagged in channel %s", channelID)
					}
				}(channelID)
			}
			tagWg.Wait()
		}
	}

	log.Println("Action: done.")
}

func runDeleteMode(s *discordgo.Session, channelID, messageID string) error {
	if err := s.ChannelMessageDelete(channelID, messageID); err != nil {
		return fmt.Errorf("failed to delete message %s: %w", messageID, err)
	}
	log.Printf("Deleted message %s from channel %s", messageID, channelID)
	return nil
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
