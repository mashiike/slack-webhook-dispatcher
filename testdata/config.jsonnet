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
