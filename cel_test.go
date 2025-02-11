package slackwebhookdispatcher

import (
	"context"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/require"
)

type celTestCase struct {
	variables *CELVariables
	expr      string
	want      bool
}

func TestCELEnv(t *testing.T) {
	env, err := NewEnv()
	require.NoError(t, err)
	tests := []celTestCase{
		{
			variables: &CELVariables{
				Payload: &slack.Msg{
					Text: "hello",
				},
			},
			expr: `payload.Text == "hello"`,
			want: true,
		},
		{
			variables: &CELVariables{
				Payload: &slack.Msg{
					Username: "Vaxila",
					Attachments: []slack.Attachment{
						{
							Color: "##ff3e4b",
							Title: "[test-server] [development] not implemented yet",
							Text:  "Occurred at 2025-01-01T23:59:59Z\nhttps://mackerel.io/orgs/example/tracing/issues/0000000000000000000000000000000000000000?issue_occurrences_id=00000",
						},
					},
				},
			},
			expr: "payload.Attachments.exists(a, a.Title.contains('[test-server]'))",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			ast, iss := env.Compile(tt.expr)
			require.NoError(t, iss.Err())
			prog, err := env.Program(ast)
			require.NoError(t, err)
			got, err := Evalute(context.Background(), prog, tt.variables)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
