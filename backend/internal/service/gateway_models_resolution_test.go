package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveEffectiveModelsFallsBackToPlatformDefaults(t *testing.T) {
	models := ResolveEffectiveModels(nil, PlatformOpenAI, GroupModelsListConfig{})
	require.Equal(t, defaultModelIDsForPlatform(PlatformOpenAI), models)
}

func TestResolveEffectiveModelsUsesAvailableModelsWhenConfigured(t *testing.T) {
	models := ResolveEffectiveModels([]string{"custom-a", "custom-b"}, PlatformOpenAI, GroupModelsListConfig{})
	require.Equal(t, []string{"custom-a", "custom-b"}, models)
}

func TestResolveEffectiveModelsAppliesCustomListAndWildcard(t *testing.T) {
	config := GroupModelsListConfig{
		Enabled: true,
		Models:  []string{"claude-sonnet-4-6", "custom-b", "missing", "custom-b"},
	}
	models := ResolveEffectiveModels([]string{"claude-sonnet-*", "custom-a", "custom-b"}, PlatformAnthropic, config)
	require.Equal(t, []string{"claude-sonnet-4-6", "custom-b"}, models)
}

func TestResolveEffectiveModelsCustomListUsesDefaultsWhenNoMappings(t *testing.T) {
	config := GroupModelsListConfig{Enabled: true, Models: []string{"gpt-5.5"}}
	models := ResolveEffectiveModels(nil, PlatformOpenAI, config)
	require.Equal(t, []string{"gpt-5.5"}, models)
}
