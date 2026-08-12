package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const statEmbedDescriptionLimit = 3800

func leaderboardRank(index int) string {
	switch index {
	case 0:
		return "🥇"
	case 1:
		return "🥈"
	case 2:
		return "🥉"
	default:
		return fmt.Sprintf("`%d.`", index+1)
	}
}

func sendPagedStatEmbedResponse(s *discordgo.Session, i *discordgo.InteractionCreate, title string, lines []string, footer string) {
	pages := buildStatEmbedPages(lines)
	if len(pages) == 0 {
		pages = []string{"No stats logged."}
	}

	for pageIndex, description := range pages {
		pageTitle := title
		pageFooter := footer
		if len(pages) > 1 {
			pageTitle = fmt.Sprintf("%s · %d/%d", title, pageIndex+1, len(pages))
			if pageFooter != "" {
				pageFooter += " • "
			}
			pageFooter += fmt.Sprintf("Page %d of %d", pageIndex+1, len(pages))
		}

		embed := &discordgo.MessageEmbed{
			Title:       pageTitle,
			Description: description,
		}
		if pageFooter != "" {
			embed.Footer = &discordgo.MessageEmbedFooter{Text: pageFooter}
		}

		if pageIndex == 0 {
			if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
			}); err != nil {
				log.Error().Err(err)
				return
			}
			continue
		}

		if _, err := s.ChannelMessageSendEmbed(i.ChannelID, embed); err != nil {
			log.Error().Err(err)
			return
		}
	}
}

func buildStatEmbedPages(lines []string) []string {
	pages := []string{}
	current := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		candidate := line
		if current != "" {
			candidate = current + "\n\n" + line
		}

		if len(candidate) > statEmbedDescriptionLimit && current != "" {
			pages = append(pages, current)
			current = line
			continue
		}
		current = candidate
	}

	if current != "" {
		pages = append(pages, current)
	}

	return pages
}
