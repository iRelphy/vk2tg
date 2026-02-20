# vk2tg — VK → Telegram bridge

Небольшой Go‑сервис, который слушает VK LongPoll и пересылает новые сообщения в Telegram‑бота всем подписчикам.

## Быстрый старт

1) Переименуй `main.env.example` в `main.env` и заполни токены:
- `VK_TOKEN`
- `TG_TOKEN`

2) Запусти:

```bash
go mod tidy
go run ./cmd/vk2tg
```

3) В Telegram напиши боту `/start` — так бот сможет начать пересылать сообщения.
Чтобы отключить пересылку — `/stop`.
