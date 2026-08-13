package main

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestCanUseModeratorCommandAllowsAdministrator(t *testing.T) {
	cfg.Discord.AdminRoleIDs = nil
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{Permissions: discordgo.PermissionAdministrator},
		},
	}

	if !canUseModeratorCommand(i) {
		t.Fatal("expected administrator to be allowed")
	}
}

func TestCanUseModeratorCommandAllowsConfiguredRole(t *testing.T) {
	cfg.Discord.AdminRoleIDs = []string{"allowed-role"}
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{Roles: []string{"other-role", "allowed-role"}},
		},
	}

	if !canUseModeratorCommand(i) {
		t.Fatal("expected configured role to be allowed")
	}
}

func TestCanUseModeratorCommandRejectsUnconfiguredRole(t *testing.T) {
	cfg.Discord.AdminRoleIDs = []string{"allowed-role"}
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{Roles: []string{"other-role"}},
		},
	}

	if canUseModeratorCommand(i) {
		t.Fatal("expected unconfigured role to be rejected")
	}
}

func TestCanUseModeratorCommandRejectsMissingMember(t *testing.T) {
	cfg.Discord.AdminRoleIDs = []string{"allowed-role"}
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{}}

	if canUseModeratorCommand(i) {
		t.Fatal("expected interaction without guild member to be rejected")
	}
}
