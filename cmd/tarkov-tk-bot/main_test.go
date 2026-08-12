package main

import (
	"os"
	"testing"
)

func TestResolveConfigPathPrefersExplicitPath(t *testing.T) {
	originalValue, hadOriginalValue := os.LookupEnv(configTOMLEnvVar)
	if err := os.Setenv(configTOMLEnvVar, "secret config"); err != nil {
		t.Fatal(err)
	}
	defer restoreEnvironmentVariable(configTOMLEnvVar, originalValue, hadOriginalValue)

	resolvedPath, cleanup, err := resolveConfigPath("explicit.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if resolvedPath != "explicit.toml" {
		t.Fatalf("expected explicit.toml, got %s", resolvedPath)
	}
}

func TestResolveConfigPathMaterializesEnvironmentConfig(t *testing.T) {
	originalValue, hadOriginalValue := os.LookupEnv(configTOMLEnvVar)
	configContent := "[discord]\nbotToken = \"test-token\"\n"
	if err := os.Setenv(configTOMLEnvVar, configContent); err != nil {
		t.Fatal(err)
	}
	defer restoreEnvironmentVariable(configTOMLEnvVar, originalValue, hadOriginalValue)

	resolvedPath, cleanup, err := resolveConfigPath("")
	if err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(resolvedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != configContent {
		t.Fatalf("unexpected config contents: %q", string(contents))
	}

	cleanup()
	if _, err := os.Stat(resolvedPath); !os.IsNotExist(err) {
		t.Fatalf("expected temporary config to be removed, got %v", err)
	}
}

func restoreEnvironmentVariable(name string, value string, hadValue bool) {
	if hadValue {
		_ = os.Setenv(name, value)
		return
	}
	_ = os.Unsetenv(name)
}
