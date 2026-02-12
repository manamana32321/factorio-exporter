package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

type DiscordChannel struct {
	session   *discordgo.Session
	channelID string
	inbound   chan InboundMessage
	botUserID string
	cfg       *Config
}

func NewDiscordChannel(token, channelID string, cfg *Config) (*DiscordChannel, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("discordgo session: %w", err)
	}

	dc := &DiscordChannel{
		session:   session,
		channelID: channelID,
		inbound:   make(chan InboundMessage, 100),
		cfg:       cfg,
	}

	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent
	session.AddHandler(dc.onMessage)

	return dc, nil
}

func (dc *DiscordChannel) Name() string { return "Discord" }

func (dc *DiscordChannel) Start(ctx context.Context) error {
	if err := dc.session.Open(); err != nil {
		return fmt.Errorf("discord open: %w", err)
	}
	dc.botUserID = dc.session.State.User.ID
	log.Printf("discord bot connected as %s", dc.session.State.User.Username)

	<-ctx.Done()
	dc.session.Close()
	return nil
}

func (dc *DiscordChannel) Send(ctx context.Context, event GameEvent) error {
	if !dc.cfg.discordEventAllowed(event.Type) {
		return nil
	}

	msg := formatGameEvent(event)
	if msg == "" {
		return nil
	}

	_, err := dc.session.ChannelMessageSend(dc.channelID, msg)
	if err != nil {
		return fmt.Errorf("send to Discord: %w", err)
	}
	return nil
}

func (dc *DiscordChannel) Messages() <-chan InboundMessage { return dc.inbound }

func (dc *DiscordChannel) Close() error {
	return dc.session.Close()
}

func (dc *DiscordChannel) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot || m.Author.ID == dc.botUserID {
		return
	}
	if m.ChannelID != dc.channelID {
		return
	}
	if m.Content == "" {
		return
	}

	author := m.Author.GlobalName
	if author == "" {
		author = m.Author.Username
	}

	dc.inbound <- InboundMessage{
		Source:  "Discord",
		Author:  author,
		Content: m.Content,
	}
}

func formatGameEvent(e GameEvent) string {
	switch e.Type {
	// Log-based events
	case "chat":
		return fmt.Sprintf("💬 **%s**: %s", e.Player, e.Message)
	case "join":
		return fmt.Sprintf("➡️ **%s** joined the game", e.Player)
	case "leave":
		return fmt.Sprintf("⬅️ **%s** left the game", e.Player)
	case "research":
		return fmt.Sprintf("🔬 Research completed: **%s**", e.Extra["tech"])
	case "rocket":
		return "🚀 **Rocket launched!**"

	// RCON-polled events
	case "research_started":
		return fmt.Sprintf("🔬 Research started: **%s**", e.Extra["name"])
	case "research_cancelled":
		return fmt.Sprintf("🔬 Research cancelled: **%s**", e.Extra["name"])
	case "player_died":
		return fmt.Sprintf("💀 **%s** died (%s)", e.Player, e.Extra["cause"])
	case "player_respawned":
		return fmt.Sprintf("🔄 **%s** respawned", e.Player)
	case "player_changed_surface":
		return fmt.Sprintf("🌍 **%s** traveled to **%s**", e.Player, e.Extra["surface"])
	case "player_promoted":
		return fmt.Sprintf("⬆️ **%s** promoted to admin", e.Player)
	case "player_demoted":
		return fmt.Sprintf("⬇️ **%s** demoted from admin", e.Player)
	case "rocket_launch_ordered":
		return "🚀 Rocket launch ordered"
	case "platform_state_changed":
		return fmt.Sprintf("🛸 Platform **%s** state changed", e.Extra["name"])
	case "cargo_ascended":
		return "📦 Cargo pod reached orbit"
	case "cargo_descended":
		return "📦 Cargo pod landed"
	case "spawner_destroyed":
		return fmt.Sprintf("🕳️ Spawner destroyed: **%s**", e.Extra["name"])
	case "surface_created":
		return fmt.Sprintf("🌍 New surface discovered: **%s**", e.Extra["name"])
	case "tag_added":
		return fmt.Sprintf("📍 Map tag added: **%s**", e.Extra["text"])

	default:
		return ""
	}
}
