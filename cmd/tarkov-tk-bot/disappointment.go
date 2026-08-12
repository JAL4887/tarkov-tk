package main

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kyleshepherd/discord-tk-bot/internal/storage"
)

func init() {
	commands = append(commands, &discordgo.ApplicationCommand{
		Name:        "disappointment",
		Description: "Log a teammate disappointment that contributed to a death",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "responsible",
				Description: "Teammate responsible for the disappointment",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "victim",
				Description: "Teammate whose death was affected",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "Reason for disappointment (Maximum 500 characters)",
				Required:    false,
			},
		},
	})
	commandHandlers["disappointment"] = handleDisappointment
}

func handleDisappointment(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	responsible := i.ApplicationCommandData().Options[0].UserValue(s)
	victim := i.ApplicationCommandData().Options[1].UserValue(s)
	reason := ""

	if len(i.ApplicationCommandData().Options) > 2 {
		reason = i.ApplicationCommandData().Options[2].StringValue()
	}
	if len(reason) > 500 {
		reason = reason[:500]
	}

	disappointment := storage.Disappointment{
		ServerID:    i.GuildID,
		Responsible: responsible.ID,
		Victim:      victim.ID,
		Reason:      reason,
		Date:        time.Now(),
	}

	_, err := store.CreateDisappointment(ctx, &disappointment)
	if err != nil {
		log.Error().Err(err)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Could not log disappointment for **%s** affecting **%s**. Please try again\n", responsible.Username, victim.Username),
			},
		})
		return
	}

	msgContent := fmt.Sprintf("Disappointment for **%s** affecting **%s** logged", responsible.Username, victim.Username)
	if disappointment.Reason != "" {
		msgContent += fmt.Sprintf(": \"**%s**\"", disappointment.Reason)
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msgContent,
		},
	})
}
