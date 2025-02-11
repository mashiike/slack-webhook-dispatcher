package slackwebhookdispatcher

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/fujiwara/ssm-lookup/ssm"
	"github.com/google/cel-go/cel"
	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	goconfig "github.com/kayac/go-config"
)

type Config struct {
	Rules []Rule `yaml:"rules"`
}

type Rule struct {
	Name        string      `yaml:"name"`
	Condition   string      `yaml:"condition"`
	Destination string      `yaml:"destination"`
	prog        cel.Program `yaml:"-"`
}

func LoadConfig(ctx context.Context, path string) (*Config, error) {
	var jsonBs []byte
	switch filepath.Ext(path) {
	case ".jsonnet":
		vm, err := makeVM(ctx)
		if err != nil {
			return nil, err
		}
		jsonStr, err := vm.EvaluateFile(path)
		if err != nil {
			return nil, err
		}
		jsonBs = []byte(jsonStr)
	case ".json":
		var err error
		jsonBs, err = os.ReadFile(path)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", filepath.Ext(path))
	}

	var config Config
	if err := goconfig.LoadWithEnvJSONBytes(&config, jsonBs); err != nil {
		return nil, err
	}
	env, err := NewEnv()
	if err != nil {
		return nil, err
	}
	for i, rule := range config.Rules {
		if rule.Name == "" {
			config.Rules[i].Name = fmt.Sprintf("rule-%d", i)
		}
		if rule.Condition == "" {
			return nil, fmt.Errorf("condition is required for rule %d", i)
		}
		if rule.Destination == "" {
			return nil, fmt.Errorf("destination is required for rule %d", i)
		}
		u, err := url.Parse(rule.Destination)
		if err != nil {
			return nil, fmt.Errorf("failed to parse destination url for rule %d: %w", i, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("unsupported scheme for rule %d: %s", i, u.Scheme)
		}
		if u.Host != "hooks.slack.com" {
			slog.Warn("config destination is not slack api, must be hooks.slack.com. reason is protection infinite loop", "rule_index", i, "destination", rule.Destination)
			return nil, fmt.Errorf("unsupported host for rule %d: %s", i, u.Host)
		}
		ast, iss := env.Compile(rule.Condition)
		if iss.Err() != nil {
			return nil, fmt.Errorf("failed to compile rule %d: %w", i, iss.Err())
		}
		prog, err := env.Program(ast)
		if err != nil {
			return nil, fmt.Errorf("failed to compile rule %d: %w", i, err)
		}
		config.Rules[i].prog = prog
	}
	return &config, nil
}

func makeVM(ctx context.Context) (*jsonnet.VM, error) {
	vm := jsonnet.MakeVM()
	for _, nf := range nativeFunctions {
		vm.NativeFunction(nf)
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	cache := &sync.Map{}
	app := ssm.New(cfg, cache)
	for _, nf := range app.JsonnetNativeFuncs(ctx) {
		vm.NativeFunction(nf)
	}
	return vm, nil
}

var nativeFunctions = []*jsonnet.NativeFunction{
	MastEnvNativeFunction,
	EnvNativeFunction,
}

var MastEnvNativeFunction = &jsonnet.NativeFunction{
	Name:   "must_env",
	Params: []ast.Identifier{"name"},
	Func: func(args []interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("must_env: invalid arguments length expected 1 got %d", len(args))
		}
		key, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("must_env: invalid arguments, expected string got %T", args[0])
		}
		val, ok := os.LookupEnv(key)
		if !ok {
			return nil, fmt.Errorf("must_env: %s not set", key)
		}
		return val, nil
	},
}
var EnvNativeFunction = &jsonnet.NativeFunction{
	Name:   "env",
	Params: []ast.Identifier{"name", "default"},
	Func: func(args []interface{}) (interface{}, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("env: invalid arguments length expected 2 got %d", len(args))
		}
		key, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("env: invalid 1st arguments, expected string got %T", args[0])
		}
		val := os.Getenv(key)
		if val == "" {
			return args[1], nil
		}
		return val, nil
	},
}
