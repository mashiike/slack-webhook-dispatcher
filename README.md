# slack-webhook-dispatcher

this app is slack incoming webhook like server,　and dispatch to slack incoming webhook with CEL(Common Expression Language) Condition.

## Usage 
Slack incoming webhook like dispacher

```
Usage of slack-webhook-dispatcher:
  -config string
        config file path (default "config.jsonnet")
  -debug
        debug mode
  -port int
        port number (default 8282)
```

### Config file

```jsonnet
local must_env = std.native('must_env');
{
  rules: [
    {
      name: 'rule1',
      condition: "payload.Attachments.exists(a, a.Title.contains('[service1]'))",
      destination: must_env('SLACK_WEBHOOK_URL_FOR_SERVICE1'),
    },
    {
      name: 'rule2',
      condition: "payload.Attachments.exists(a, a.Title.contains('[service2]'))",
      destination: must_env('SLACK_WEBHOOK_URL_FOR_SERVICE2'),
    },
  ],
}
```

if incoming webhook paylaod is following, 

```json
{
    "username":"Test",
    "attachments":[
        {
            "color":"#ff3e4b",
            "title":"[service1] [development] test exception",
            "text":"Occurred at 2025-01-01T25:59:59Z"
        }
    ]
}
```

then, this app dispatch to `SLACK_WEBHOOK_URL_FOR_SERVICE1` slack incoming webhook.

## License 

MIT
