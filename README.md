# vk2tg — VK → Telegram bridge

Небольшой Go‑сервис, который слушает VK LongPoll и пересылает новые сообщения в Telegram‑бота всем подписчикам.

## Быстрый старт

1) Скопируй `main.env.example` в `main.env` и заполни токены:
- `VK_TOKEN`
- `TG_TOKEN`

2) Запусти:

```bash
go mod tidy
go run ./cmd/vk2tg
```

3) В Telegram напиши боту `/start` в нужном чате (личка/группа) — этот чат станет подписчиком.
Чтобы отключить — `/stop`.

## Структура проекта

- `cmd/vk2tg` — точка входа (минимальный main)
- `internal/app` — сборка всех компонентов и запуск
- `internal/config` — env и конфиг
- `internal/vk` — VK API + резолверы (имя/название чата)
- `internal/tg` — Telegram отправка + команды
- `internal/storage` — subscribers.json
- `internal/bridge` — логика склейки VK→TG
- `internal/util` — маленькие утилиты (HTML escape, лимиты, время)

## Важно

- `main.env` и `subscribers.json` игнорируются через `.gitignore`.
- Telegram форматирование: мы используем HTML parse mode и всегда экранируем пользовательский текст.
