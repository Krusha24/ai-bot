package tg

import (
	"ai-bot/internal/domain"
	"context"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type Bot struct {
	bot *telego.Bot
}

func NewTgBot(token string) (*Bot, error) {
	client, err := telego.NewBot(token)
	if err != nil {
		return nil, err
	}
	return &Bot{bot: client}, nil
}

func (b *Bot) SendMessage(message string, chatID int64) {
	ctx := context.Background()
	b.bot.SendMessage(ctx,
		tu.Message(
			tu.ID(chatID),
			message,
		),
	)
}

func (b *Bot) StartListening(events chan<- domain.Event) {
	ctx := context.Background()
	updates, _ := b.bot.UpdatesViaLongPolling(ctx, nil)
	for update := range updates {
		if update.Message != nil {
			event := domain.Event{Type: "message", Payload: update.Message.Text, ChatID: update.Message.Chat.ID}
			events <- event
		}
	}
}
