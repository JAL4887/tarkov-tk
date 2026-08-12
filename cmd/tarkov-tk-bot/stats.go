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

	commandHandlers["tkstats"] = handleTKStatsPresentation
	commandHandlers["disappointmentstats"] = handleDisappointmentStats
	commandHandlers["stats"] = handleCombinedStats
}

func handleTKStatsPresentation(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	members, err := s.GuildMembers(i.GuildID, "", 1000)
	if err != nil {
		respondStatsError(s, i, "Could not get Discord members for server. Please try again.", err)
		return
	}

	var killer *discordgo.User
	var shouldCSV bool
	for _, option := range i.ApplicationCommandData().Options {
		switch option.Type {
		case discordgo.ApplicationCommandOptionUser:
			killer = option.UserValue(s)
		case discordgo.ApplicationCommandOptionBoolean:
			shouldCSV = option.BoolValue()
		}
	}

	var kills []*storage.Kill
	if killer != nil {
		kills, err = store.ListPlayerKillsForServer(ctx, i.GuildID, killer.ID)
	} else {
		kills, err = store.ListKillsForServer(ctx, i.GuildID)
	}
	if err != nil {
		respondStatsError(s, i, "Could not get kills for server. Please try again.", err)
		return
	}

	if killer == nil {
		sortKillsHighestToLowest(kills)
	}

	displayNames := memberDisplayNames(members)
	displayKills := make([]*storage.Kill, 0, len(kills))
	for _, kill := range kills {
		copyOfKill := *kill
		copyOfKill.Killer = displayNameForID(kill.Killer, displayNames)
		copyOfKill.Victim = displayNameForID(kill.Victim, displayNames)
		displayKills = append(displayKills, &copyOfKill)
	}

	guildName := guildDisplayName(s, i.GuildID)
	if shouldCSV {
		csvBuffer := &bytes.Buffer{}
		if err := gocsv.Marshal(displayKills, csvBuffer); err != nil {
			respondStatsError(s, i, "Could not generate TK CSV. Please try again.", err)
			return
		}

		fileName := fmt.Sprintf("%s-TarkovTKStats-%s.csv", guildName, time.Now().Format("2006-01-02"))
		if killer != nil {
			fileName = fmt.Sprintf("%s-%s-TarkovTKStats-%s.csv", killer.Username, guildName, time.Now().Format("2006-01-02"))
		}

		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Successfully generated server stats",
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

	title := fmt.Sprintf("💀 Team Kill History · %s", guildName)
	if killer != nil {
		title = fmt.Sprintf("💀 Team Kill History · %s", displayNameForID(killer.ID, displayNames))
	}

	lines := make([]string, 0, len(displayKills))
	for _, kill := range displayKills {
		line := fmt.Sprintf("**%s → %s**  •  %s", kill.Killer, kill.Victim, kill.Date.Format("01/02/2006"))
		if kill.Reason != "" {
			line += fmt.Sprintf("\n> %s", kill.Reason)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, "No team kills logged.")
	}

	sendPagedStatEmbedResponse(s, i, title, lines, fmt.Sprintf("%d team kill(s)", len(displayKills)))
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

	if responsible == nil {
		sortDisappointmentsHighestToLowest(disappointments)
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

	title := fmt.Sprintf("😞 Disappointments · %s", guildName)
	if responsible != nil {
		title = fmt.Sprintf("😞 Disappointments · %s", displayNameForID(responsible.ID, displayNames))
	}

	lines := make([]string, 0, len(displayDisappointments))
	for _, disappointment := range displayDisappointments {
		line := fmt.Sprintf("**%s → %s**  •  %s", disappointment.Responsible, disappointment.Victim, disappointment.Date.Format("01/02/2006"))
		if disappointment.Reason != "" {
			line += fmt.Sprintf("\n> %s", disappointment.Reason)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, "No disappointments logged.")
	}

	sendPagedStatEmbedResponse(s, i, title, lines, fmt.Sprintf("%d disappointment(s)", len(displayDisappointments)))
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
	title := fmt.Sprintf("📊 TK + Disappointment Stats · %s", guildName)

	stats := make([]*CombinedPlayerStats, 0, len(statsByPlayer))
	if selectedPlayer != nil {
		playerStats, ok := statsByPlayer[selectedPlayer.ID]
		if !ok {
			playerStats = &CombinedPlayerStats{PlayerID: selectedPlayer.ID}
		}
		stats = append(stats, playerStats)
		title = fmt.Sprintf("📊 Combined Stats · %s", displayNameForID(selectedPlayer.ID, displayNames))
	} else {
		for _, playerStats := range statsByPlayer {
			stats = append(stats, playerStats)
		}
		sortCombinedPlayerStatsHighestToLowest(stats, displayNames)
	}

	lines := make([]string, 0, len(stats))
	for index, playerStats := range stats {
		prefix := ""
		if selectedPlayer == nil {
			prefix = leaderboardRank(index) + " "
		}
		lines = append(lines, fmt.Sprintf(
			"%s**%s**\n`TK %d`  `TD %d`  `DIS %d`  `REC %d`",
			prefix,
			displayNameForID(playerStats.PlayerID, displayNames),
			playerStats.TeamKills,
			playerStats.TeamDeaths,
			playerStats.Disappointments,
			playerStats.DisappointmentsReceived,
		))
	}
	if len(lines) == 0 {
		lines = append(lines, "No TK or disappointment stats logged.")
	}

	footer := "TK = Team Kills • TD = Team Deaths • DIS = Disappointments • REC = Received"
	if selectedPlayer == nil {
		footer = fmt.Sprintf("%d player(s) • %s", len(stats), footer)
	}
	sendPagedStatEmbedResponse(s, i, title, lines, footer)
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

func sortCombinedPlayerStatsHighestToLowest(stats []*CombinedPlayerStats, displayNames map[string]string) {
	sort.SliceStable(stats, func(i, j int) bool {
		if stats[i].TeamKills != stats[j].TeamKills {
			return stats[i].TeamKills > stats[j].TeamKills
		}
		if stats[i].Disappointments != stats[j].Disappointments {
			return stats[i].Disappointments > stats[j].Disappointments
		}
		if stats[i].TeamDeaths != stats[j].TeamDeaths {
			return stats[i].TeamDeaths > stats[j].TeamDeaths
		}
		if stats[i].DisappointmentsReceived != stats[j].DisappointmentsReceived {
			return stats[i].DisappointmentsReceived > stats[j].DisappointmentsReceived
		}

		left := strings.ToLower(displayNameForID(stats[i].PlayerID, displayNames))
		right := strings.ToLower(displayNameForID(stats[j].PlayerID, displayNames))
		if left == right {
			return stats[i].PlayerID < stats[j].PlayerID
		}
		return left < right
	})
}

func sortDisappointmentsHighestToLowest(disappointments []*storage.Disappointment) {
	counts := make(map[string]int)
	for _, disappointment := range disappointments {
		counts[disappointment.Responsible]++
	}

	sort.SliceStable(disappointments, func(i, j int) bool {
		leftCount := counts[disappointments[i].Responsible]
		rightCount := counts[disappointments[j].Responsible]
		if leftCount != rightCount {
			return leftCount > rightCount
		}
		if !disappointments[i].Date.Equal(disappointments[j].Date) {
			return disappointments[i].Date.After(disappointments[j].Date)
		}
		if disappointments[i].Responsible != disappointments[j].Responsible {
			return disappointments[i].Responsible < disappointments[j].Responsible
		}
		return disappointments[i].Victim < disappointments[j].Victim
	})
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
