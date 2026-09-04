package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestParseOpenAIImageModelSizeAlias(t *testing.T) {
	for _, tc := range []struct {
		name     string
		model    string
		wantBase string
		wantSize string
		wantOK   bool
	}{
		{"1k alias", "gpt-image-2-1k", "gpt-image-2", "1024x1024", true},
		{"2k alias", "gpt-image-2-2k", "gpt-image-2", "2048x2048", true},
		{"4k alias", "gpt-image-2-4k", "gpt-image-2", "4096x4096", true},
		{"uppercase alias", "gpt-image-2-4K", "gpt-image-2", "4096x4096", true},
		{"alias on 1.5", "gpt-image-1.5-2k", "gpt-image-1.5", "2048x2048", true},
		{"plain image model", "gpt-image-2", "gpt-image-2", "", false},
		{"dotted version is not an alias", "gpt-image-1.5", "gpt-image-1.5", "", false},
		{"base must stay an image model", "gpt-image-1k", "gpt-image-1k", "", false},
		{"unknown suffix", "gpt-image-2-8k", "gpt-image-2-8k", "", false},
		{"non image model", "gpt-5.6", "gpt-5.6", "", false},
		{"empty", "", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base, size, ok := ParseOpenAIImageModelSizeAlias(tc.model)
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.wantBase, base)
			require.Equal(t, tc.wantSize, size)
		})
	}
}

func TestExpandOpenAIImageModelSizeAlias(t *testing.T) {
	t.Run("rewrites model and injects size", func(t *testing.T) {
		body := []byte(`{"model":"gpt-image-2-2k","input":"draw a cat"}`)
		out, base, size, ok := ExpandOpenAIImageModelSizeAlias(body)
		require.True(t, ok)
		require.Equal(t, "gpt-image-2", base)
		require.Equal(t, "2048x2048", size)
		require.Equal(t, "gpt-image-2", gjson.GetBytes(out, "model").String())
		require.Equal(t, "2048x2048", gjson.GetBytes(out, "size").String())
		require.Equal(t, "draw a cat", gjson.GetBytes(out, "input").String())
	})

	t.Run("explicit size wins over the alias", func(t *testing.T) {
		body := []byte(`{"model":"gpt-image-2-4k","size":"1024x1024"}`)
		out, base, size, ok := ExpandOpenAIImageModelSizeAlias(body)
		require.True(t, ok)
		require.Equal(t, "gpt-image-2", base)
		require.Empty(t, size, "client size must not be overwritten")
		require.Equal(t, "1024x1024", gjson.GetBytes(out, "size").String())
	})

	t.Run("non alias body is untouched", func(t *testing.T) {
		body := []byte(`{"model":"gpt-image-2","input":"x"}`)
		out, _, _, ok := ExpandOpenAIImageModelSizeAlias(body)
		require.False(t, ok)
		require.Equal(t, body, out)
	})

	t.Run("non json body is untouched", func(t *testing.T) {
		body := []byte("--boundary\r\nContent-Disposition: form-data; name=\"model\"\r\n\r\ngpt-image-2-2k\r\n")
		out, _, _, ok := ExpandOpenAIImageModelSizeAlias(body)
		require.False(t, ok)
		require.Equal(t, body, out)
	})
}
