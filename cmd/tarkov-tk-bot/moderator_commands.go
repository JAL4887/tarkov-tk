package main

import "github.com/bwmarrin/discordgo"

func init() {
	delete(commandHandlers, "tkinfo")
	delete(commandHandlers, "tkthanks")
	wrapModeratorCommand("tkreset")
	wrapModeratorCommand("tkremove")
}

func wrapModeratorCommand(name string) {
	handler, ok := commandHandlers[name]
	if !ok {
		return
	}
	commandHandlers[name] = func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if !canUseModeratorCommand(i) {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You do not have permission to use this command.",
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		handler(s, i)
	}
}

func canUseModeratorCommand(i *discordgo.InteractionCreate) bool {
	if i == nil || i.Member == nil {
		return false
	}
	if i.Member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}
	for _, memberRoleID := range i.Member.Roles {
		for _, allowedRoleID := range cfg.Discord.AdminRoleIDs {
			if memberRoleID == allowedRoleID && allowedRoleID != "" {
				return true
			}
		}
	}
	return false
}
