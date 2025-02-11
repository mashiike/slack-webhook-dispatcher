package slackwebhookdispatcher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("SLACK_WEBHOOK_URL_FOR_SERVICE1", "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX")
	t.Setenv("SLACK_WEBHOOK_URL_FOR_SERVICE2", "https://hooks.slack.com/services/T00000000/B00000000/YYYYYYYYYYYYYYYYYYYYYYYY")

	testConfigs := []string{
		"testdata/config.json",
		"testdata/config.jsonnet",
	}
	for _, path := range testConfigs {
		t.Run(path, func(t *testing.T) {
			config, err := LoadConfig(context.Background(), path)
			require.NoError(t, err)
			require.Len(t, config.Rules, 2)

		})
	}
}
