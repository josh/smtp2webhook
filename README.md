# smtp2webhook

SMTP to Webhook Relay.

```yml
version: "3"
services:
  smtp2webhook:
    restart: always
    image: ghcr.io/josh/smtp2webhook
    environment:
      - DOMAIN=example.com
      - CODE=d039b5
      - WEBHOOK_TEST=https://d039b5.requestcatcher.com/test
```

Will forward mail to `d039b5+test@example.com` to `https://d039b5.requestcatcher.com/test`.
