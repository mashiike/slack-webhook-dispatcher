package slackwebhookdispatcher

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"
	"github.com/slack-go/slack"
)

type CELVariables struct {
	Payload *slack.Msg `json:"payload" cel:"payload"`
	TeamID  string     `json:"team_id" cel:"teamId"`
	BotID   string     `json:"bot_id" cel:"botId"`
	Token   string     `json:"token" cel:"token"`
}

func NewEnv() (*cel.Env, error) {
	opts := make([]cel.EnvOption, 0)
	opts = append(opts, VariableOptionsFromObject("payload", slack.Msg{})...)
	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func Evalute(ctx context.Context, prog cel.Program, variables *CELVariables) (bool, error) {
	out, _, err := prog.ContextEval(ctx, map[string]interface{}{
		"payload": variables.Payload,
		"teamId":  variables.TeamID,
		"botId":   variables.BotID,
		"token":   variables.Token,
	})
	if err != nil {
		return false, err
	}
	got, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("unexpected type: %T", out.Value())
	}
	return got, nil
}

func VariableOptionsFromObject(variableName string, v any) []cel.EnvOption {
	rt := reflect.TypeOf(v)
	var pkgPath string
	paths := strings.Split(rt.PkgPath(), "/")
	if len(paths) != 0 {
		pkgPath = paths[len(paths)-1]
	}
	objectName := fmt.Sprintf("%s.%s", pkgPath, rt.Name())
	return []cel.EnvOption{
		cel.Variable(variableName, cel.ObjectType(objectName)),
		ext.NativeTypes(rt, ext.ParseStructTags(true)),
	}
}
