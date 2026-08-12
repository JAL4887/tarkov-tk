package main

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gocarina/gocsv"
	"github.com/kyleshepherd/discord-tk-bot/internal/storage"
)

type CombinedPlayerStats struct {
	PlayerID                string
	TeamKills               int
	TeamDeaths              int
	Disappointments         int
	DisappointmentsReceived int
}

func init() {
	commands = append(commands,
		&discordgo.ApplicationCommand{
			Name:        "disappointmentstats",
			Description: "Get disappointment history for the server or a single player",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "responsible",
					Description: "User whose disappointments to retrieve",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "csv",
					Description: "Generate CSV file of stats",
					Required:    false,
				},
			},
		},
		&discordgo.ApplicationCommand{
			Name:        "stats",
			Description: "Show combined TK and disappointment stats",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "player",
					Description: "Show combined stats for a single player",
					Required:    false,
				},
			},
		},
	)

	commandHandlers["disappointmentstats"] = handleDisappointmentStats
	commandHandlers["stats"] = handleCombinedStats
}

func handleDisappointmentStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	members, err := s.GuildMembers(i.GuildID, "", 1000)
	if err != nil {
		respondStatsError(s, i, "Could not get Discord members for server. Please try again.", err)
		return
	}

	var responsible *discordgo.User
	var shouldCSV bool
	for _, option := range i.ApplicationCommandData().Options {
		switch option.Type {
		case discordgo.ApplicationCommandOptionUser:
			responsible = option.UserValue(s)
		case discordgo.ApplicationCommandOptionBoolean:
			shouldCSV = option.BoolValue()
		}
	}

	var disappointments []*storage.Disappointment
	if responsible != nil {
		disappointments, err = store.ListPlayerDisappointmentsForServer(ctx, i.GuildID, responsible.ID)
	} else {
		disappointments, err = store.ListDisappointmentsForServer(ctx, i.GuildID)
	}
	if err != nil {
		respondStatsError(s, i, "Could not get disappointments for server. Please try again.", err)
		return
	}

	displayNames := memberDisplayNames(members)
	displayDisappointments := make([]*storage.Disappointment, 0, len(disappointments))
	for _, disappointment := range disappointments {
		copyOfDisappointment := *disappointment
		copyOfDisappointment.Responsible = displayNameForID(disappointment.Responsible, displayNames)
		copyOfDisappointment.Victim = displayNameForID(disappointment.Victim, displayNames)
		displayDisappointments = append(displayDisappointments, &copyOfDisappointment)
	}

	guildName := guildDisplayName(s, i.GuildID)
	if shouldCSV {
		csvBuffer := &bytes.Buffer{}
		if err := gocsv.Marshal(displayDisappointments, csvBuffer); err != nil {
			respondStatsError(s, i, "Could not generate disappointment CSV. Please try again.", err)
			return
		}

		fileName := fmt.Sprintf("%s-TarkovDisappointmentStats-%s.csv", guildName, time.Now().Format("2006-01-02"))
		if responsible != nil {
			fileName = fmt.Sprintf("%s-%s-TarkovDisappointmentStats-%s.csv", responsible.Username, guildName, time.Now().Format("2006-01-02"))
		}

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Successfully generated disappointment stats",
				Files: []*discordgo.File{
					{
						Name:        fileName,
						ContentType: "text/csv",
						Reader:      csvBuffer,
					},
				},
			},
		})
		if err != nil {
			log.Error().Err(err)
		}
		return
	}

	header := fmt.Sprintf("**Disappointment Stats for %s**\n\n", guildName)
	if responsible != nil {
		header = fmt.Sprintf("**Disappointment Stats for %s**\n\n", displayNameForID(responsible.ID, displayNames))
	}

	lines := make([]string, 0, len(displayDisappointments))
	for _, disappointment := range displayDisappointments {
		line := fmt.Sprintf("%s - **%s** disappointed **%s**", disappointment.Date.Format("01/02/2006"), disappointment.Responsible, disappointment.Victim)
		if disappointment.Reason != "" {
			line += fmt.Sprintf(": \"%s\"", disappointment.Reason)
		}
		lines = append(lines, line+"\n\n")
	}
	if len(lines) == 0 {
		lines = append(lines, "No disappointments logged.\n")
	}

	sendPagedInteractionResponse(s, i, header, lines)
}

func handleCombinedStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	kills, err := store.ListKillsForServer(ctx, i.GuildID)
	if err != nil {
		respondStatsError(s, i, "Could not get TK stats for server. Please try again.", err)
		return
	}

	disappointments, err := store.ListDisappointmentsForServer(ctx, i.GuildID)
	if err != nil {
		respondStatsError(s, i, "Could not get disappointment stats for server. Please try again.", err)
		return
	}

	members, err := s.GuildMembers(i.GuildID, "", 1000)
	if err != nil {
		respondStatsError(s, i, "Could not get Discord members for server. Please try again.", err)
		return
	}

	var selectedPlayer *discordgo.User
	for _, option := range i.ApplicationCommandData().Options {
		if option.Type == discordgo.ApplicationCommandOptionUser {
			selectedPlayer = option.UserValue(s)
		}
	}

	statsByPlayer := buildCombinedPlayerStats(kills, disappointments)
	displayNames := memberDisplayNames(members)
	guildName := guildDisplayName(s, i.GuildID)
	header := fmt.Sprintf("**Combined TK + Disappointment Stats for %s**\n\n", guildName)

	stats := make([]*CombinedPlayerStats, 0, len(statsByPlayer))
	if selectedPlayer != nil {
		playerStats, ok := statsByPlayer[selectedPlayer.ID]
		if !ok {
			playerStats = &CombinedPlayerStats{PlayerID: selectedPlayer.ID}
		}
		stats = append(stats, playerStats)
		header = fmt.Sprintf("**Combined Stats for %s**\n\n", displayNameForID(selectedPlayer.ID, displayNames))
	} else {
		for _, playerStats := range statsByPlayer {
			stats = append(stats, playerStats)
		}
		sort.Slice(stats, func(i, j int) bool {
			left := strings.ToLower(displayNameForID(stats[i].PlayerID, displayNames))
			right := strings.ToLower(displayNameForID(stats[j].PlayerID, displayNames))
			if left == right {
				return stats[i].PlayerID < stats[j].PlayerID
			}
			return left < right
		})
	}

	lines := make([]string, 0, len(stats))
	for _, playerStats := range stats {
		lines = append(lines, fmt.Sprintf(
			"**%s** - TKs: %d | Team Deaths: %d | Disappointments: %d | Received: %d\n",
			displayNameForID(playerStats.PlayerID, displayNames),
			playerStats.TeamKills,
			playerStats.TeamDeaths,
			playerStats.Disappointments,
			playerStats.DisappointmentsReceived,
		))
	}
	if len(lines) == 0 {
		lines = append(lines, "No TK or disappointment stats logged.\n")
	}

	sendPagedInteractionResponse(s, i, header, lines)
}

func buildCombinedPlayerStats(kills []*storage.Kill, disappointments []*storage.Disappointment) map[string]*CombinedPlayerStats {
	stats := map[string]*CombinedPlayerStats{}

	ensurePlayer := func(playerID string) *CombinedPlayerStats {
		if playerID == "" {
			return nil
		}
		playerStats, ok := stats[playerID]
		if !ok {
			playerStats = &CombinedPlayerStats{PlayerID: playerID}
			stats[playerID] = playerStats
		}
		return playerStats
	}

	for _, kill := range kills {
		if playerStats := ensurePlayer(kill.Killer); playerStats != nil {
			playerStats.TeamKills++
		}
		if playerStats := ensurePlayer(kill.Victim); playerStats != nil {
			playerStats.TeamDeaths++
		}
	}

	for _, disappointment := range disappointments {
		if playerStats := ensurePlayer(disappointment.Responsible); playerStats != nil {
			playerStats.Disappointments++
		}
		if playerStats := ensurePlayer(disappointment.Victim); playerStats != nil {
			playerStats.DisappointmentsReceived++
		}
	}

	return stats
}

func memberDisplayNames(members []*discordgo.Member) map[string]string {
	displayNames := make(map[string]string, len(members))
	for _, member := range members {
		if member == nil || member.User == nil {
			continue
		}
		if member.Nick != "" {
			displayNames[member.User.ID] = member.Nick
		} else {
			displayNames[member.User.ID] = member.User.Username
		}
	}
	return displayNames
}

func displayNameForID(playerID string, displayNames map[string]string) string {
	if displayName, ok := displayNames[playerID]; ok && displayName != "" {
		return displayName
	}
	return playerID
}

func guildDisplayName(s *discordgo.Session, guildID string) string {
	guild, err := s.Guild(guildID)
	if err == nil && guild != nil && guild.Name != "" {
		return guild.Name
	}
	return guildID
}

func sendPagedInteractionResponse(s *discordgo.Session, i *discordgo.InteractionCreate, header string, lines []string) {
	pages := []string{}
	current := header
	for _, line := range lines {
		if len(current)+len(line) > 1900 && current != "" {
			pages = append(pages, current)
			current = ""
		}
		current += line
	}
	if current != "" {
		pages = append(pages, current)
	}
	if len(pages) == 0 {
		pages = append(pages, header)
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: pages[0]},
	}); err != nil {
		log.Error().Err(err)
		return
	}

	for _, page := range pages[1:] {
		if _, err := s.ChannelMessageSend(i.ChannelID, page); err != nil {
			log.Error().Err(err)
			return
		}
	}
}

func respondStatsError(s *discordgo.Session, i *discordgo.InteractionCreate, message string, err error) {
	log.Error().Err(err)
	if responseErr := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: message},
	}); responseErr != nil {
		log.Error().Err(responseErr)
	}
}
