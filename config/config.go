package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Answer struct {
	Text          string `yaml:"text"`
	Emoji         string `yaml:"emoji"`
	EmojiName     string `yaml:"emoji_name"`
	EmojiID       string `yaml:"emoji_id"`
	EmojiAnimated bool   `yaml:"emoji_animated"`
}

type Poll struct {
	Name             string   `yaml:"name"`
	Label            string   `yaml:"label"`
	DurationHours    int      `yaml:"duration_hours"`
	AllowMultiselect bool     `yaml:"allow_multiselect"`
	Answers          []Answer `yaml:"answers"`
}

type Schedule struct {
	Poll      string `yaml:"poll"`
	ChannelID string `yaml:"channel_id"`
	Title     string `yaml:"title"` // supports {date} placeholder → DD.MM of the firing day
	Day       string `yaml:"day"`   // monday … sunday
	At        string `yaml:"at"`    // HH:MM in the configured timezone
}

type RootSchedule struct {
	Poll  string `yaml:"poll"`
	Title string `yaml:"title"` // supports {date} placeholder → DD.MM of the firing day
	Day   string `yaml:"day"`   // monday … sunday
	At    string `yaml:"at"`    // HH:MM in the configured timezone
}

type MainSchedule struct {
	Enabled   bool   `yaml:"enabled"`
	ChannelID string `yaml:"channel_id"`
}

type Channel struct {
	GuildID          string       `yaml:"team_guild_id"`
	TeamRoleID       string       `yaml:"team_role_id"`
	TaggingEmojiId   string       `yaml:"tagging_emoji_id"`
	TaggingEmojiName string       `yaml:"tagging_emoji_name"`
	PickDay          string       `yaml:"pick_day"` // day of week this squad's polls are posted, e.g. "wednesday"
	MainSchedule     MainSchedule `yaml:"main_schedule"`
	Polls            []Poll       `yaml:"polls"`
	Schedules        []Schedule   `yaml:"schedules"`
}

type Config struct {
	Timezone  string             `yaml:"timezone"` // e.g. "Europe/Moscow"
	Schedules []RootSchedule     `yaml:"schedules"`
	Channels  map[string]Channel `yaml:"channels"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "UTC"
	}
	return &cfg, nil
}

// FindPoll searches a channel's polls by name.
func (ch Channel) FindPoll(name string) (Poll, error) {
	for _, p := range ch.Polls {
		if p.Name == name {
			return p, nil
		}
	}
	return Poll{}, fmt.Errorf("poll %q not found", name)
}

// FindPoll searches all channels for a poll by name, returning the poll and its channel.
func (cfg *Config) FindPoll(name string) (Poll, Channel, error) {
	for _, ch := range cfg.Channels {
		for _, p := range ch.Polls {
			if p.Name == name {
				return p, ch, nil
			}
		}
	}
	return Poll{}, Channel{}, fmt.Errorf("poll %q not found in config", name)
}

// AllGuildIDs returns the unique guild IDs across all channels.
func (cfg *Config) AllGuildIDs() []string {
	seen := map[string]bool{}
	var ids []string
	for _, ch := range cfg.Channels {
		if ch.GuildID != "" && !seen[ch.GuildID] {
			seen[ch.GuildID] = true
			ids = append(ids, ch.GuildID)
		}
	}
	return ids
}

// AllPolls returns deduplicated polls across all channels (first occurrence wins).
func (cfg *Config) AllPolls() []Poll {
	seen := map[string]bool{}
	var polls []Poll
	for _, ch := range cfg.Channels {
		for _, p := range ch.Polls {
			if !seen[p.Name] {
				seen[p.Name] = true
				polls = append(polls, p)
			}
		}
	}
	return polls
}
