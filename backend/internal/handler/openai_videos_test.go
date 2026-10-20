package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidVideoTaskID(t *testing.T) {
	for _, id := range []string{"video_123", "task-123", "20261020_123.abc"} {
		require.True(t, validVideoTaskID(id), id)
	}
	for _, id := range []string{"", ".", "..", "../models", "a/b", "a?x=1", "a%2fb", "a#frag"} {
		require.False(t, validVideoTaskID(id), id)
	}
}
