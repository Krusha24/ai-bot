package main

import (
	"ai-bot/internal/brain"
	"ai-bot/internal/domain"
	"ai-bot/internal/storage"
	"ai-bot/internal/tg"

	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	ollama "github.com/ollama/ollama/api"
)

const SYSTEMPROMPT = "Ты — мой дерзкий и саркастичный цифровой клон (или выбери любой другой характер). Ты общаешься с другом в Telegram. Никогда не говори, что ты ИИ. Общайся строго на русском языке, используй сленг. Твой ответ ВСЕГДА должен быть строго в формате JSON: {\"type_action\": \"reply\" или \"ignore\", \"text\": \"твой ответ\"}"

func buildContext(dp *storage.DB, vectorDB *storage.ChromemDB, client *brain.Brain, chatID int64, userMessage, extraSystemMsg string) ([]ollama.Message, error) {
	activeHistory, err := dp.GetActiveHistory(chatID)
	if err != nil {
		log.Printf("Проблема получение активных сообщений для чата %d: %v", chatID, err)
		return nil, err
	}
	summary, err := dp.GetLatestSummary(chatID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		log.Printf("Проблема получение саммари для чата %d: %v", chatID, err)
		return nil, err
	}

	var finalSystemContent, memoryString string
	finalSystemContent = SYSTEMPROMPT
	if userMessage != "" {
		queryEmbedding, err := client.GetEmbedding(userMessage)
		if err != nil {
			log.Printf("Проблема получение eмбендингов по сообщению для чата %d: %v", chatID, err)
		}
		chunks, err := vectorDB.SearchSimilar(context.Background(), chatID, queryEmbedding, 3)
		if err != nil {
			log.Printf("Проблема получение похожих eмбендингов по сообщению для чата %d: %v", chatID, err)
		} else if len(chunks) > 0 {
			for _, chunk := range chunks {
				memoryString += chunk.Content + "\n"
			}
			finalSystemContent = SYSTEMPROMPT + "\n\nДолгосрочная память (возможно, релевантные факты из прошлых бесед):\n" + memoryString
		}
	}

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

func handleMessageEvent(event domain.Event, dp *storage.DB, vectorDB *storage.ChromemDB, client *brain.Brain, telegramBot *tg.Bot) {
	userMessage := event.Payload
	dp.SaveMessage(event.ChatID, "user", userMessage)

	promptForLLM, err := buildContext(dp, vectorDB, client, event.ChatID, userMessage, "")
	if err != nil {
		return
	}

	action, err := client.Think(promptForLLM)
	if err != nil {
		log.Printf("Проблема генарции ответа для чата %d: %v", event.ChatID, err)
		return
	}
	if action.Type == "reply" {
		telegramBot.SendMessage(action.Text, event.ChatID)
		dp.SaveMessage(event.ChatID, "assistant", action.Text)
	}

	activeHistory, err := dp.GetActiveHistory(event.ChatID)
	if err != nil {
		log.Printf("Проблема получение активных сообщений для чата %d: %v", event.ChatID, err)
		return
	}
	if len(activeHistory) > 15 {
		summary, err := client.Summarize(activeHistory)
		if err != nil {
			log.Printf("Проблема саммарайза сообщений для чата %d: %v", event.ChatID, err)
			return
		}

		embedding, err := client.GetEmbedding(summary)
		if err != nil {
			log.Printf("Проблема получение эмбединга для чата %d: %v", event.ChatID, err)
			return
		}

		chunk := domain.MemoryChunk{
			Id:        fmt.Sprintf("%d_%d", event.ChatID, time.Now().Unix()),
			ChatID:    event.ChatID,
			Content:   summary,
			Embedding: embedding,
		}
		vectorDB.SaveChunk(context.Background(), chunk)

		dp.SaveSummary(event.ChatID, summary)
		dp.ArchiveOldMessages(event.ChatID, 15)
	}
}

func userProcessing(dp *storage.DB, vectorDB *storage.ChromemDB, chatID int64, client *brain.Brain, telegramBot *tg.Bot) {
	messages, err := dp.GetActiveHistory(chatID)
	if err != nil {
		log.Printf("Проблема получение активных сообщений для чата %d: %v", chatID, err)
		return
	}

	var lastUserMessage string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastUserMessage = messages[i].Content
			break
		}
	}

	prompt, err := buildContext(dp, vectorDB, client, chatID, lastUserMessage, "Системное уведомление: Прошло некоторое время с вашего последнего общения. У тебя есть своя жизнь, свои дела и увлечения (ты мог залипать в YouTube, кодить, играть или просто спать). Посмотри на контекст диалога выше. Появилось ли у тебя желание написать этому человеку первым? Поделиться рандомной мыслью, скинуть мемную фразу или спросить, как дела? Если тебе лень, ты занят или диалог не располагает к продолжению — смело выбирай 'ignore'. Если хочешь написать — выбирай 'reply' и пиши в своем обычном стиле.")
	if err != nil {
		return
	}
	action, err := client.Think(prompt)
	if err != nil {
		log.Printf("Проблема генарции ответа для чата %d: %v", chatID, err)
		return
	}
	if action.Type == "reply" {
		telegramBot.SendMessage(action.Text, chatID)
		dp.SaveMessage(chatID, "assistant", action.Text)
	}
}

func handleTimerEvent(dp *storage.DB, vectorDB *storage.ChromemDB, client *brain.Brain, telegramBot *tg.Bot) {
	activeChats, err := dp.GetActiveChats()
	if err != nil {
		log.Printf("Проблема получения активных чатов для функции таймера: %v", err)
		return
	}
	for _, chatID := range activeChats {
		go userProcessing(dp, vectorDB, chatID, client, telegramBot)
	}
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

	vectorDB, err := storage.NewChromemDB()
	if err != nil {
		log.Fatal(err)
	}

	telegramBot, err := tg.NewTgBot(os.Getenv("BOTTOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	client, err := brain.NewBrain("qwen2.5", "nomic-embed-text")
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
			go handleMessageEvent(event, dp, vectorDB, client, telegramBot)

		case "timer":
			go handleTimerEvent(dp, vectorDB, client, telegramBot)
		}

	}
}
