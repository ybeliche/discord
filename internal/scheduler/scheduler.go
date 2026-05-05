package scheduler

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
	"github.com/ybeliche/discord/config"
	"github.com/ybeliche/discord/internal/poll"
)

var weekdays = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

// NextOccurrence returns the next date (including today) for the given day name in loc.
func NextOccurrence(day string, loc *time.Location) (time.Time, error) {
	target, ok := weekdays[strings.ToLower(day)]
	if !ok {
		return time.Time{}, fmt.Errorf("unknown day %q", day)
	}
	now := time.Now().In(loc)
	daysUntil := int(target - now.Weekday())
	if daysUntil < 0 {
		daysUntil += 7
	}
	return now.AddDate(0, 0, daysUntil), nil
}

// buildCron converts "friday" + "22:00" → "0 22 * * 5"
func buildCron(day, at string) (string, error) {
	wd, ok := weekdays[strings.ToLower(day)]
	if !ok {
		return "", fmt.Errorf("unknown day %q", day)
	}
	parts := strings.SplitN(at, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid time %q, expected HH:MM", at)
	}
	return fmt.Sprintf("%s %s * * %d", parts[1], parts[0], int(wd)), nil
}

// Start runs the cron scheduler, firing polls on their configured schedule.
// Returns the cron instance so the caller can Stop() it on shutdown.
func Start(s *discordgo.Session, cfg *config.Config) *cron.Cron {
	if len(cfg.Schedules) == 0 {
		return nil
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Printf("Scheduler: invalid timezone %q, falling back to UTC: %v", cfg.Timezone, err)
		loc = time.UTC
	}

	c := cron.New(cron.WithLocation(loc))

	for _, sched := range cfg.Schedules {
		p, err := cfg.FindPoll(sched.Poll)
		if err != nil {
			log.Printf("Scheduler: skipping %q: %v", sched.Poll, err)
			continue
		}
		if sched.ChannelID == "" {
			log.Printf("Scheduler: skipping %q: channel_id not set", sched.Poll)
			continue
		}
		expr, err := buildCron(sched.Day, sched.At)
		if err != nil {
			log.Printf("Scheduler: skipping %q: %v", sched.Poll, err)
			continue
		}

		if _, err := c.AddFunc(expr, func() {
			date := time.Now().In(loc).Format("02.01")
			title := strings.ReplaceAll(sched.Title, "{date}", date)
			log.Printf("Scheduler: posting %q title %q", sched.Poll, title)
			if err := poll.Post(s, sched.ChannelID, title, p); err != nil {
				log.Printf("Scheduler: failed to post %q: %v", sched.Poll, err)
				return
			}
			if cfg.TeamRoleID != "" {
				if _, err := s.ChannelMessageSend(sched.ChannelID, fmt.Sprintf("<@&%s>", cfg.TeamRoleID)); err != nil {
					log.Printf("Scheduler: failed to tag team for %q: %v", sched.Poll, err)
				}
			}
		}); err != nil {
			log.Printf("Scheduler: failed to register cron for %q: %v", sched.Poll, err)
			continue
		}
		log.Printf("Scheduler: %q → every %s at %s %s (cron: %s)", sched.Poll, sched.Day, sched.At, cfg.Timezone, expr)
	}

	c.Start()
	return c
}
