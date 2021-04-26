# smtp2webhook

SMTP to Webhook Relay.

```yml
version: "3"
services:
  smtp2webhook:
    restart: always
    image: ghcr.io/josh/smtp2webhook
    ports:
      - "25:25"
    environment:
      - SMTP2WEBHOOK_DOMAIN=example.com
      - SMTP2WEBHOOK_CODE=d039b5
      - SMTP2WEBHOOK_URL_TEST=https://d039b5.requestcatcher.com/test
```

Will forward mail to `d039b5+test@example.com` to `https://d039b5.requestcatcher.com/test`.
