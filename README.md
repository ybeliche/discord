# Alabama

A Discord bot that posts attendance polls for gaming events. It runs in two modes:

- **Bot mode** — persistent process with a `/questionnaire` slash command and a cron scheduler that fires polls automatically on a configured schedule.
- **Action mode** — short-lived run (used in GitHub Actions) that posts all scheduled polls once and exits.

---

## How it works

### Poll templates

Polls are defined in `config/config.yaml`. Each poll has a name, a display label, a duration, multiselect flag, and a list of answers. Each answer can carry a Discord emoji (Unicode or custom server emoji by ID).

### Schedules

Each schedule entry ties a poll template to a channel, a weekday, a time, and a title. The `{date}` placeholder in the title is replaced with the actual date (`DD.MM`) at fire time.

### Two run modes

| Env var | Value | Behavior |
|---|---|---|
| `GITHUB_ACTIONS` | `true` | Action mode: posts all polls once using the next upcoming date for each schedule, then exits |
| *(unset)* | | Bot mode: registers slash commands, starts cron scheduler, runs until interrupted |

---

## Configuration

`config/config.yaml`:

```yaml
timezone: "Europe/Moscow"      # IANA timezone for schedule firing and date formatting
team_role_id: "123456789"      # Role to mention when posting a poll (leave empty to skip)

polls:
  - name: friday_event         # Internal identifier, referenced by schedules
    label: "Friday Event"      # Shown in the /questionnaire dropdown
    duration_hours: 24
    allow_multiselect: false
    answers:
      - text: "Coming"
        emoji: "✅"            # Unicode emoji
      - text: "Late"
        emoji_name: "custom_emoji_name"   # Custom server emoji (by name)
        emoji_id: "123456789012345678"    # Custom server emoji ID
        emoji_animated: false             # true for animated emoji
      - text: "Not coming"
        emoji: "❌"

schedules:
  - poll: friday_event          # Must match a poll name above
    channel_id: "111222333"     # Channel to post in
    title: "Friday Event ({date}) at 22:00 MSK!"   # {date} → DD.MM on fire day
    day: friday                 # sunday–saturday
    at: "22:00"                 # HH:MM in the configured timezone
```

---

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `DISCORD_TOKEN` | Yes | Bot token from the Discord Developer Portal |
| `GUILD_ID` | Yes | ID of the Discord server to register slash commands on |
| `GITHUB_ACTIONS` | No | Set to `true` to run in action mode |

---

## Running locally

```sh
export DISCORD_TOKEN=your_token
export GUILD_ID=your_guild_id
go run .
```

The bot registers the `/questionnaire` slash command on startup, starts the cron scheduler, and waits. Press `Ctrl+C` to stop gracefully.

---

## Docker

```sh
docker build -t alabama .
docker run --rm \
  -e DISCORD_TOKEN=your_token \
  -e GUILD_ID=your_guild_id \
  alabama
```

The image is a two-stage build: Go builder on `golang:alpine`, runtime on `alpine:3.21` with `ca-certificates` and `tzdata` for timezone support.

---

## GitHub Actions

The workflow in `.github/workflows/docker.yml` runs every Wednesday at 09:00 UTC (`0 9 * * 3`) and can also be triggered manually via `workflow_dispatch`.

It uses the composite action at `.github/actions/run-bot/`, which builds and runs the Docker image with `GITHUB_ACTIONS=true`. In this mode the bot calculates the next occurrence of each scheduled weekday, posts all polls, and exits.

Required repository secrets:

| Secret | Description |
|---|---|
| `DISCORD_TOKEN` | Discord bot token |
| `GUILD_ID` | Discord server ID |

---

## Project structure

```
main.go                        # Entry point, bot/action mode dispatch
config/
  config.go                    # Config struct and YAML loader
  config.yaml                  # Poll templates and schedules
internal/
  commands/
    registry.go                # Slash command registration
    questionnaire.go           # /questionnaire interaction handler
  poll/
    poll.go                    # Builds and posts a Discord poll
  scheduler/
    scheduler.go               # Cron scheduler and date helpers
.github/
  workflows/docker.yml         # Scheduled CI workflow
  actions/run-bot/action.yml   # Composite action (Docker run)
Dockerfile
```

---

## Bot permissions

The bot requires the following Discord permissions:

- **Send Messages** — to post polls in channels
- **Use Application Commands** — to register and respond to slash commands

Intents used: `GUILDS`, `GUILD_MESSAGES`.
