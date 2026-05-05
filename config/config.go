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

type Config struct {
	Timezone     string     `yaml:"timezone"` // e.g. "Europe/Moscow"
	TeamRoleID   string     `yaml:"team_role_id"`
	TaggingEmoji string     `yaml:"tagging_emoji"`
	Polls        []Poll     `yaml:"polls"`
	Schedules    []Schedule `yaml:"schedules"`
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

func (cfg *Config) FindPoll(name string) (*Poll, error) {
	for i := range cfg.Polls {
		if cfg.Polls[i].Name == name {
			return &cfg.Polls[i], nil
		}
	}
	return nil, fmt.Errorf("poll %q not found in config", name)
}
