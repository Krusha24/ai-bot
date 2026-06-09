package main

import (
	"ai-bot/internal/brain"
	"ai-bot/internal/domain"
	"ai-bot/internal/storage"
	"ai-bot/internal/tg"

	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	ollama "github.com/ollama/ollama/api"
)

const SYSTEMPROMPT = "Ты — мой дерзкий и саркастичный цифровой клон (или выбери любой другой характер). Ты общаешься с другом в Telegram. Никогда не говори, что ты ИИ. Общайся строго на русском языке, используй сленг. Твой ответ ВСЕГДА должен быть строго в формате JSON: {\"type_action\": \"reply\" или \"ignore\", \"text\": \"твой ответ\"}"

func buildContext(dp *storage.DB, chatID int64, extraSystemMsg string) ([]ollama.Message, error) {
	activeHistory, err := dp.GetActiveHistory(chatID)
	if err != nil {
		log.Printf("Описание проблемы: %v", err)
		return nil, err
	}
	summary, err := dp.GetLatestSummary(chatID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		log.Printf("Ошибка получения саммари: %v", err)
		return nil, err
	}

	finalSystemContent := SYSTEMPROMPT
	if summary != "" {
		finalSystemContent += "\n\nКонтекст прошлых бесед: " + summary
	}

	systemMsg := ollama.Message{Role: "system", Content: finalSystemContent}

	prompt := append([]ollama.Message{systemMsg}, activeHistory...)
	if extraSystemMsg != "" {
		prompt = append(prompt, ollama.Message{Role: "system", Content: extraSystemMsg})
	}
	return prompt, nil
}

func main() {
	err := godotenv.Load("../../config/.env")
	if err != nil {
		log.Fatal("Ошибка при загрузке файла .env")
	}

	dp, err := storage.NewDB("bot.db")
	if err != nil {
		log.Fatal(err)
	}

	telegramBot, err := tg.NewTgBot(os.Getenv("BOTTOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	client, err := brain.NewOllamaClient()
	if err != nil {
		log.Fatal(err)
	}
	events := make(chan domain.Event)

	go func() {
		for {
			time.Sleep(4 * time.Minute)
			event := domain.Event{Type: "timer", Payload: ""}
			events <- event
		}

	}()

	go telegramBot.StartListening(events)
	for event := range events {

		switch event.Type {
		case "message":

			dp.SaveMessage(event.ChatID, "user", event.Payload)

			promptForLLM, err := buildContext(dp, event.ChatID, "")
			if err != nil {
				log.Printf("Описание проблемы: %v", err)
				continue
			}

			action, err := client.Think(promptForLLM)
			if err != nil {
				log.Printf("Описание проблемы: %v", err)
				continue
			}
			if action.Type == "reply" {
				telegramBot.SendMessage(action.Text, event.ChatID)
			}
			dp.SaveMessage(event.ChatID, "assistant", action.Text)

			activeHistory, err := dp.GetActiveHistory(event.ChatID)
			if err != nil {
				log.Printf("Описание проблемы: %v", err)
				continue
			}
			if len(activeHistory) > 15 {
				summary, err := client.Summarize(activeHistory)
				if err != nil {
					log.Printf("Описание проблемы: %v", err)
					continue
				}
				dp.SaveSummary(event.ChatID, summary)
				dp.ArchiveOldMessages(event.ChatID, 15)
			}
		case "timer":
			activeChats, err := dp.GetActiveChats()
			if err != nil {
				log.Printf("Описание проблемы: %v", err)
				continue
			}
			for _, chatID := range activeChats {
				prompt, err := buildContext(dp, chatID, "Системное уведомление: Прошло некоторое время с вашего последнего общения. У тебя есть своя жизнь, свои дела и увлечения (ты мог залипать в YouTube, кодить, играть или просто спать). Посмотри на контекст диалога выше. Появилось ли у тебя желание написать этому человеку первым? Поделиться рандомной мыслью, скинуть мемную фразу или спросить, как дела? Если тебе лень, ты занят или диалог не располагает к продолжению — смело выбирай 'ignore'. Если хочешь написать — выбирай 'reply' и пиши в своем обычном стиле.")

				action, err := client.Think(prompt)
				if err != nil {
					log.Printf("Описание проблемы: %v", err)
					continue
				}
				if action.Type == "reply" {
					telegramBot.SendMessage(action.Text, chatID)
					dp.SaveMessage(chatID, "assistant", action.Text)
				}
			}
		}

	}
}
