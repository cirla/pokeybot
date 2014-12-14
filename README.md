pokeybot
========

[Pokey the Penguin](http://yellow5.com/pokey/archive/) comic Slack integration (Slash Command + Incoming Webhooks)

Type "/pokey" in slack for a random Pokey comic!

Runs on Heroku + Postgres

Dependencies
------------

Requires the [Heroku go buildpack](https://github.com/kr/heroku-buildpack-go.git)

`heroku create -b https://github.com/kr/heroku-buildpack-go.git`

Requires [godep](https://github.com/kr/godep)

`go get github.com/kr/godep`

`godep save`

Requires the following environment variables:

| Environment Variable  | Description                                                       | Example                                             |
|-------------------------|---------------------------------------------------------------------|-------------------------------------------------------|---|---|
| DATABASE_URL            | URL for Postgres database                                           | postgres://abc:xyz@example.com:1234/mydatabase        |
| PGSSL                   | Set to "require" on Heroku                                          | require                                               |
| SECRET_KEY_POKEY        | Secret key required for API calls that make changes to the database | abc123!@#                                             |
| SLACK_TOKEN_POKEY       | Slack token for the Slash Command integration                       | def456$%^                                             |
| SLACK_WEBHOOK_URL_POKEY | Slack Incoming Webhooks URL                                         | https://hooks.slack.com/services/abc123/def456/xyz789 |

Requires custom Slack emojis :pokey:, :littlegirl:, :mrnutty:, and :skeptopotamus:
