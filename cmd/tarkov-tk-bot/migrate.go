package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/kyleshepherd/discord-tk-bot/internal/migration"
	"github.com/kyleshepherd/discord-tk-bot/internal/storage"
	"github.com/kyleshepherd/discord-tk-bot/internal/storage/firestore"
	"github.com/spf13/cobra"
)

type legacyMigrationOptions struct {
	CSVPath     string
	GuildID     string
	MappingFile string
	Commit      bool
}

type legacyResolution struct {
	Record         migration.LegacyRecord
	KillerID       string
	VictimID       string
	Fingerprint    string
	Disappointment bool
	Duplicate      bool
}

func migrateLegacyCmd() *cobra.Command {
	options := legacyMigrationOptions{}
	cmd := &cobra.Command{
		Use:   "migrate-legacy",
		Short: "Validate or import a legacy Tarkov TK CSV export",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLegacyMigration(options)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&options.CSVPath, "csv", "", "path to the legacy Tarkov TK CSV export")
	flags.StringVar(&options.GuildID, "guild-id", "", "Discord guild ID (defaults to discord.guildID from config)")
	flags.StringVar(&options.MappingFile, "mapping-file", "", "optional CSV with LegacyName,DiscordID columns")
	flags.BoolVar(&options.Commit, "commit", false, "write validated records to Firestore; without this flag the command is a dry run")
	return cmd
}

func runLegacyMigration(options legacyMigrationOptions) error {
	if strings.TrimSpace(options.CSVPath) == "" {
		return fmt.Errorf("legacy CSV path is required; pass --csv")
	}

	guildID := strings.TrimSpace(options.GuildID)
	if guildID == "" {
		guildID = strings.TrimSpace(cfg.Discord.GuildID)
	}
	if guildID == "" {
		return fmt.Errorf("guild ID is required; set discord.guildID in config or pass --guild-id")
	}
	if strings.TrimSpace(cfg.Discord.BotToken) == "" {
		return fmt.Errorf("discord bot token is required to resolve legacy usernames")
	}

	file, err := os.Open(options.CSVPath)
	if err != nil {
		return fmt.Errorf("open legacy CSV: %w", err)
	}
	defer file.Close()

	records, err := migration.ParseLegacyCSV(file)
	if err != nil {
		return err
	}

	manualMappings, err := loadLegacyMappings(options.MappingFile)
	if err != nil {
		return err
	}

	discordSession, err := discordgo.New("Bot " + cfg.Discord.BotToken)
	if err != nil {
		return fmt.Errorf("create Discord session: %w", err)
	}

	members, err := listAllGuildMembers(discordSession, guildID)
	if err != nil {
		return fmt.Errorf("list Discord guild members: %w", err)
	}

	resolver := buildLegacyMemberResolver(members)
	resolved := make([]legacyResolution, 0, len(records))
	unresolved := map[string]string{}

	for _, record := range records {
		killerID, killerErr := resolveLegacyName(record.Killer, manualMappings, resolver)
		if killerErr != nil {
			unresolved[record.Killer] = killerErr.Error()
		}
		victimID, victimErr := resolveLegacyName(record.Victim, manualMappings, resolver)
		if victimErr != nil {
			unresolved[record.Victim] = victimErr.Error()
		}

		resolved = append(resolved, legacyResolution{
			Record:         record,
			KillerID:       killerID,
			VictimID:       victimID,
			Fingerprint:    migration.Fingerprint(record),
			Disappointment: migration.IsDisappointment(record.Reason),
		})
	}

	printUnresolvedUsers(unresolved)
	if len(unresolved) > 0 {
		printLegacyMigrationSummary(resolved, 0, len(unresolved), false)
		return fmt.Errorf("legacy migration has unresolved Discord users; provide --mapping-file entries before importing")
	}

	ctx := context.Background()
	store, err := firestore.NewKillStore(ctx, cfg.Firebase.ProjectID, cfg.Firebase.ServiceAccountFilePath)
	if err != nil {
		return fmt.Errorf("open Firestore: %w", err)
	}
	defer store.Close()

	duplicateCount := 0
	for i := range resolved {
		var exists bool
		if resolved[i].Disappointment {
			exists, err = store.LegacyDisappointmentExists(ctx, resolved[i].Fingerprint)
		} else {
			exists, err = store.LegacyKillExists(ctx, resolved[i].Fingerprint)
		}
		if err != nil {
			return fmt.Errorf("check legacy record duplicate: %w", err)
		}
		resolved[i].Duplicate = exists
		if exists {
			duplicateCount++
		}
	}

	printLegacyMigrationSummary(resolved, duplicateCount, 0, options.Commit)
	if !options.Commit {
		fmt.Println("Dry run complete. No records were written. Re-run with --commit after reviewing this report.")
		return nil
	}

	importedKills := 0
	importedDisappointments := 0
	skipped := 0
	for _, item := range resolved {
		if item.Duplicate {
			skipped++
			continue
		}

		if item.Disappointment {
			created, err := store.ImportLegacyDisappointment(ctx, &storage.Disappointment{
				ServerID:    guildID,
				Responsible: item.KillerID,
				Victim:      item.VictimID,
				Reason:      item.Record.Reason,
				Date:        item.Record.Date,
			}, item.Fingerprint)
			if err != nil {
				return fmt.Errorf("import legacy disappointment: %w", err)
			}
			if created {
				importedDisappointments++
			} else {
				skipped++
			}
			continue
		}

		created, err := store.ImportLegacyKill(ctx, &storage.Kill{
			ServerID: guildID,
			Killer:   item.KillerID,
			Victim:   item.VictimID,
			Reason:   item.Record.Reason,
			Date:     item.Record.Date,
		}, item.Fingerprint)
		if err != nil {
			return fmt.Errorf("import legacy kill: %w", err)
		}
		if created {
			importedKills++
		} else {
			skipped++
		}
	}

	fmt.Printf("Imported team kills: %d\n", importedKills)
	fmt.Printf("Imported disappointments: %d\n", importedDisappointments)
	fmt.Printf("Skipped existing records: %d\n", skipped)
	return nil
}

func loadLegacyMappings(path string) (map[string]string, error) {
	mappings := map[string]string{}
	if strings.TrimSpace(path) == "" {
		return mappings, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open mapping file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read mapping file: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("mapping file is empty")
	}

	header := map[string]int{}
	for i, value := range rows[0] {
		header[strings.ToLower(strings.TrimSpace(value))] = i
	}
	legacyIndex, hasLegacy := header["legacyname"]
	discordIndex, hasDiscord := header["discordid"]
	if !hasLegacy || !hasDiscord {
		return nil, fmt.Errorf("mapping file must include LegacyName and DiscordID columns")
	}

	for rowIndex, row := range rows[1:] {
		if legacyIndex >= len(row) || discordIndex >= len(row) {
			return nil, fmt.Errorf("mapping row %d is missing required columns", rowIndex+2)
		}
		legacyName := normalizeLegacyName(row[legacyIndex])
		discordID := strings.TrimSpace(row[discordIndex])
		if legacyName == "" || discordID == "" {
			return nil, fmt.Errorf("mapping row %d must include both LegacyName and DiscordID", rowIndex+2)
		}
		mappings[legacyName] = discordID
	}

	return mappings, nil
}

func listAllGuildMembers(session *discordgo.Session, guildID string) ([]*discordgo.Member, error) {
	members := []*discordgo.Member{}
	after := ""
	for {
		batch, err := session.GuildMembers(guildID, after, 1000)
		if err != nil {
			return nil, err
		}
		members = append(members, batch...)
		if len(batch) < 1000 {
			break
		}
		after = batch[len(batch)-1].User.ID
	}
	return members, nil
}

func buildLegacyMemberResolver(members []*discordgo.Member) map[string][]string {
	resolver := map[string][]string{}
	for _, member := range members {
		if member == nil || member.User == nil {
			continue
		}
		addResolverCandidate(resolver, member.User.Username, member.User.ID)
		if member.Nick != "" {
			addResolverCandidate(resolver, member.Nick, member.User.ID)
		}
	}
	return resolver
}

func addResolverCandidate(resolver map[string][]string, name string, userID string) {
	key := normalizeLegacyName(name)
	if key == "" {
		return
	}
	for _, existingID := range resolver[key] {
		if existingID == userID {
			return
		}
	}
	resolver[key] = append(resolver[key], userID)
}

func resolveLegacyName(name string, manualMappings map[string]string, resolver map[string][]string) (string, error) {
	key := normalizeLegacyName(name)
	if manualID, ok := manualMappings[key]; ok {
		return manualID, nil
	}

	matches := resolver[key]
	if len(matches) == 0 {
		return "", fmt.Errorf("no Discord member matches %q", name)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple Discord members match %q", name)
	}
	return matches[0], nil
}

func normalizeLegacyName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func printUnresolvedUsers(unresolved map[string]string) {
	if len(unresolved) == 0 {
		return
	}

	names := make([]string, 0, len(unresolved))
	for name := range unresolved {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Println("Unresolved users:")
	for _, name := range names {
		fmt.Printf("  %s: %s\n", name, unresolved[name])
	}
}

func printLegacyMigrationSummary(resolved []legacyResolution, duplicateCount int, unresolvedCount int, commit bool) {
	teamKills := 0
	disappointments := 0
	for _, item := range resolved {
		if item.Disappointment {
			disappointments++
		} else {
			teamKills++
		}
	}

	mode := "DRY RUN"
	if commit {
		mode = "COMMIT"
	}
	fmt.Printf("Legacy migration mode: %s\n", mode)
	fmt.Printf("Source records: %d\n", len(resolved))
	fmt.Printf("Team kills: %d\n", teamKills)
	fmt.Printf("Disappointments: %d\n", disappointments)
	fmt.Printf("Existing duplicates: %d\n", duplicateCount)
	fmt.Printf("Unresolved users: %d\n", unresolvedCount)
}
