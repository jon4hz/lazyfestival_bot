package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
)

type Client struct {
	bandsByDay [3][]Band
	bot        *gotgbot.Bot
	alerts     map[int64][]*Alert
	mu         sync.Mutex
}

func NewClient(token string, bandsByDay [3][]Band) (*Client, error) {
	b, err := gotgbot.NewBot(token, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	alerts, err := loadAlerts()
	if err != nil {
		return nil, fmt.Errorf("failed to load alerts: %w", err)
	}

	return &Client{
		bandsByDay: bandsByDay,
		bot:        b,
		alerts:     alerts,
	}, nil
}

func (c *Client) Run() error {
	// Create updater and dispatcher.
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		// If an error is returned by a handler, log it and continue going.
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Println("an error occurred while handling update:", err.Error())
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})
	updater := ext.NewUpdater(dispatcher, nil)

	dispatcher.AddHandler(handlers.NewCommand("start", c.startHandler))
	dispatcher.AddHandler(handlers.NewCommand("today", c.todayHandler))
	dispatcher.AddHandler(handlers.NewCommand("timetable", c.timetableHandler))
	dispatcher.AddHandler(handlers.NewCommand("alerts", c.alertsHandler))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("day_"), c.daysHandler))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("band_"), c.bandsHandler))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("alert_"), c.manageAlertsHandler))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("back"), c.backHandler))

	err := updater.StartPolling(c.bot, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 9,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 10,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start polling: %w", err)
	}
	log.Printf("%s has been started...\n", c.bot.User.Username)

	go c.alertLoop()

	updater.Idle()
	return nil
}

func (c *Client) alertLoop() {
	ticker := time.NewTicker(3 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		for chatID, alerts := range c.alerts {
			for _, alert := range alerts {
				if time.Now().After(alert.Time) {
					continue
				}
				if alert.Min5 && time.Now().After(alert.Time.Add(-5*time.Minute)) {
					_, err := c.bot.SendMessage(chatID, fmt.Sprintf("🔔 5 minutes until %q", alert.Band), nil)
					if err != nil {
						log.Println("failed to send message:", err)
					}
					alert.Min5 = false
					log.Printf("5min alert for %q sent to %d", alert.Band, chatID)
				}
				if alert.Min15 && time.Now().After(alert.Time.Add(-15*time.Minute)) {
					_, err := c.bot.SendMessage(chatID, fmt.Sprintf("🔔 15 minutes until %q", alert.Band), nil)
					if err != nil {
						log.Println("failed to send message:", err)
					}
					alert.Min15 = false
					log.Printf("15min alert for %q sent to %d", alert.Band, chatID)
				}
				if alert.Min30 && time.Now().After(alert.Time.Add(-30*time.Minute)) {
					_, err := c.bot.SendMessage(chatID, fmt.Sprintf("🔔 30 minutes until %q", alert.Band), nil)
					if err != nil {
						log.Println("failed to send message:", err)
					}
					alert.Min30 = false
					log.Printf("30min alert for %q sent to %d", alert.Band, chatID)
				}
				if alert.Hour1 && time.Now().After(alert.Time.Add(-1*time.Hour)) {
					_, err := c.bot.SendMessage(chatID, fmt.Sprintf("🔔 1 hour until %q", alert.Band), nil)
					if err != nil {
						log.Println("failed to send message:", err)
					}
					alert.Hour1 = false
					log.Printf("1h alert for %q sent to %d", alert.Band, chatID)
				}
				if alert.Hour2 && time.Now().After(alert.Time.Add(-2*time.Hour)) {
					_, err := c.bot.SendMessage(chatID, fmt.Sprintf("🔔 2 hours until %q", alert.Band), nil)
					if err != nil {
						log.Println("failed to send message:", err)
					}
					alert.Hour2 = false
					log.Printf("2h alert for %q sent to %d", alert.Band, chatID)
				}
			}
		}
		go func() {
			if err := storeAlerts(c.alerts); err != nil {
				log.Println("failed to store alerts:", err)
			}
		}()
		c.mu.Unlock()
	}
}

func (c *Client) startHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	_, err := b.SendMessage(ctx.EffectiveChat.Id, "👋 Hey,\nUse /today to get todays timetable\nUse /alerts to manage your alerts", nil)
	if err != nil {
		return err
	}
	_, err = b.SetMyCommands([]gotgbot.BotCommand{
		{Command: "today", Description: "Get todays timetable"},
		{Command: "timetable", Description: "Get the timetable"},
		{Command: "alerts", Description: "Manage your alerts"},
	}, nil)
	return err
}

func (c *Client) todayHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	var i int
	switch time.Now().Day() {
	case c.bandsByDay[0][0].Date.Day():
		i = 0
	case c.bandsByDay[1][0].Date.Day():
		i = 1
	case c.bandsByDay[2][0].Date.Day():
		i = 2
	default:
		_, err := b.SendMessage(ctx.EffectiveChat.Id, "No bands today 😞", nil)
		return err
	}

	var msg strings.Builder
	msg.WriteString("📅 ")
	msg.WriteString(c.bandsByDay[i][0].Date.Format("Monday, January 02"))
	msg.WriteString("\n")
	msg.WriteString("--------------------")
	msg.WriteString("\n")

	for j, band := range c.bandsByDay[i] {
		// check if band is currently playing
		if j < len(c.bandsByDay[i])-1 && time.Now().After(band.Date) && time.Now().Before(c.bandsByDay[i][j+1].Date) {
			msg.WriteString("🎸 ")
		} else if j == len(c.bandsByDay[i])-1 && time.Now().After(band.Date) && time.Now().Before(band.Date.Add(1*time.Hour)) {
			msg.WriteString("🎸 ")
		} else {
			msg.WriteString("      ")
		}

		msg.WriteString(band.Date.Format("15:04"))
		msg.WriteString("  ")
		msg.WriteString(band.Name)
		msg.WriteString(" (")
		msg.WriteByte(band.Stage[0])
		msg.WriteString(")\n")
	}

	_, err := b.SendMessage(ctx.EffectiveChat.Id, msg.String(), &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})
	return err
}

func (c *Client) timetableHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	var msg strings.Builder
	for _, bands := range c.bandsByDay {
		msg.WriteString("📅 ")
		msg.WriteString(bands[0].Date.Format("Monday, January 02"))
		msg.WriteString("\n")
		msg.WriteString("--------------------")
		msg.WriteString("\n")

		for j, band := range bands {
			// check if band is currently playing
			if j < len(bands)-1 && time.Now().After(band.Date) && time.Now().Before(bands[j+1].Date) {
				msg.WriteString("🎸 ")
			} else if j == len(bands)-1 && time.Now().After(band.Date) && time.Now().Before(band.Date.Add(1*time.Hour)) {
				msg.WriteString("🎸 ")
			} else {
				msg.WriteString("      ")
			}

			msg.WriteString(band.Date.Format("15:04"))
			msg.WriteString("  ")
			msg.WriteString(band.Name)
			msg.WriteString(" (")
			msg.WriteByte(band.Stage[0])
			msg.WriteString(")\n")
		}
		msg.WriteString("\n")
	}

	_, err := b.SendMessage(ctx.EffectiveChat.Id, msg.String(), &gotgbot.SendMessageOpts{
		ParseMode: "HTML",
	})
	return err
}

func (c *Client) alertsHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	_, err := ctx.EffectiveMessage.Reply(b, "📅 For which day would you like to manage your alerts?", &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: festivalDays(),
		},
	})
	return err
}

func (c *Client) backHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	cb := ctx.Update.CallbackQuery
	_, err := cb.Answer(b, nil)
	if err != nil {
		return err
	}

	_, _, err = cb.Message.EditText(b, "📅 For which day would you like to manage your alerts?", &gotgbot.EditMessageTextOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: festivalDays(),
		},
	})
	return err
}

func festivalDays() [][]gotgbot.InlineKeyboardButton {
	days := [3]string{"Thursday, June 13", "Friday, June 14", "Saturday, June 15"}
	var festivalDays [][]gotgbot.InlineKeyboardButton
	for _, day := range days {
		festivalDays = append(festivalDays, []gotgbot.InlineKeyboardButton{
			{Text: day, CallbackData: "day_" + day},
		})
	}
	return festivalDays
}

func (c *Client) daysHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	cb := ctx.Update.CallbackQuery
	_, err := cb.Answer(b, nil)
	if err != nil {
		return err
	}

	day := strings.TrimPrefix(cb.Data, "day_")
	var i int
	switch day {
	case "Thursday, June 13":
		i = 0
	case "Friday, June 14":
		i = 1
	case "Saturday, June 15":
		i = 2
	default:
		i = 0
	}

	_, _, err = cb.Message.EditText(b, "🤘 Select the band.", &gotgbot.EditMessageTextOpts{
		ParseMode: "HTML",
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: bandAlerts(c.bandsByDay[i]),
		},
	})
	return err
}

func bandAlerts(bands []Band) [][]gotgbot.InlineKeyboardButton {
	var bandAlerts [][]gotgbot.InlineKeyboardButton
	for _, band := range bands {
		bandAlerts = append(bandAlerts, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("%s (%s)", band.Name, band.Date.Format("15:04")), CallbackData: "band_" + band.Name},
		})
	}
	return append(bandAlerts, []gotgbot.InlineKeyboardButton{{Text: "🔙 Back", CallbackData: "back"}})
}

func (c *Client) bandsHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	cb := ctx.Update.CallbackQuery
	_, err := cb.Answer(b, nil)
	if err != nil {
		return err
	}

	bandName := strings.TrimPrefix(cb.Data, "band_")
	chatID := ctx.EffectiveChat.Id

	alert := c.alertsByChatIDAndBandName(chatID, bandName)

	_, _, err = cb.Message.EditText(b, fmt.Sprintf("🔔 Trigger alerts for %q", bandName), &gotgbot.EditMessageTextOpts{
		ParseMode: "HTML",
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: alertButtons(alert),
		},
	})
	return err
}

func (c *Client) alertsByChatIDAndBandName(chatID int64, bandName string) *Alert {
	c.mu.Lock()
	defer c.mu.Unlock()
	alerts, ok := c.alerts[chatID]
	if !ok {
		return &Alert{
			Band: bandName,
		}
	}
	for _, alert := range alerts {
		if alert.Band == bandName {
			return alert
		}
	}
	return &Alert{
		Band: bandName,
	}
}

func alertButtons(alert *Alert) [][]gotgbot.InlineKeyboardButton {
	var alertButtons [][]gotgbot.InlineKeyboardButton
	topRow := make([]gotgbot.InlineKeyboardButton, 0, 1)
	middleRow := make([]gotgbot.InlineKeyboardButton, 0, 2)
	bottomRow := make([]gotgbot.InlineKeyboardButton, 0, 2)
	if alert.Min5 {
		topRow = append(topRow, gotgbot.InlineKeyboardButton{
			Text: "✅ 5 min", CallbackData: buildAlertCB("5min", "d", alert.Band),
		})
	} else {
		topRow = append(topRow, gotgbot.InlineKeyboardButton{
			Text: "⬜ 5 min", CallbackData: buildAlertCB("5min", "a", alert.Band),
		})
	}

	if alert.Min15 {
		middleRow = append(middleRow, gotgbot.InlineKeyboardButton{
			Text: "✅ 15 min", CallbackData: buildAlertCB("15min", "d", alert.Band),
		})
	} else {
		middleRow = append(middleRow, gotgbot.InlineKeyboardButton{
			Text: "⬜ 15 min", CallbackData: buildAlertCB("15min", "a", alert.Band),
		})
	}
	if alert.Min30 {
		middleRow = append(middleRow, gotgbot.InlineKeyboardButton{
			Text: "✅ 30 min", CallbackData: buildAlertCB("30min", "d", alert.Band),
		})
	} else {
		middleRow = append(middleRow, gotgbot.InlineKeyboardButton{
			Text: "⬜ 30 min", CallbackData: buildAlertCB("30min", "a", alert.Band),
		})
	}

	if alert.Hour1 {
		bottomRow = append(bottomRow, gotgbot.InlineKeyboardButton{
			Text: "✅ 1 hour", CallbackData: buildAlertCB("1hour", "d", alert.Band),
		})
	} else {
		bottomRow = append(bottomRow, gotgbot.InlineKeyboardButton{
			Text: "⬜ 1 hour", CallbackData: buildAlertCB("1hour", "a", alert.Band),
		})
	}
	if alert.Hour2 {
		bottomRow = append(bottomRow, gotgbot.InlineKeyboardButton{
			Text: "✅ 2 hours", CallbackData: buildAlertCB("2hours", "d", alert.Band),
		})
	} else {
		bottomRow = append(bottomRow, gotgbot.InlineKeyboardButton{
			Text: "⬜ 2 hours", CallbackData: buildAlertCB("2hours", "a", alert.Band),
		})
	}
	alertButtons = append(alertButtons, topRow, middleRow, bottomRow, []gotgbot.InlineKeyboardButton{{Text: "🔙 Back", CallbackData: "back"}})
	return alertButtons
}

func buildAlertCB(frame string, action string, band string) string {
	return fmt.Sprintf("alert_%s_%s_%s", action, frame, band)
}

func (c *Client) manageAlertsHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	cb := ctx.Update.CallbackQuery
	_, err := cb.Answer(b, nil)
	if err != nil {
		return err
	}

	parts := strings.Split(strings.TrimPrefix(cb.Data, "alert_"), "_")
	action := parts[0]
	frame := parts[1]
	band := parts[2]
	chatID := ctx.EffectiveChat.Id

	alert := c.alertsByChatIDAndBandName(chatID, band)

	enable := action == "a"
	switch frame {
	case "5min":
		alert.Min5 = enable
	case "15min":
		alert.Min15 = enable
	case "30min":
		alert.Min30 = enable
	case "1hour":
		alert.Hour1 = enable
	case "2hours":
		alert.Hour2 = enable
	}

	c.setAlert(chatID, band, alert)

	_, _, err = cb.Message.EditText(b, fmt.Sprintf("🔔 Trigger alerts for %q", band), &gotgbot.EditMessageTextOpts{
		ParseMode: "HTML",
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: alertButtons(alert),
		},
	})
	return err
}

func (c *Client) setAlert(chatID int64, band string, alert *Alert) {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer func() {
		go func() {
			if err := storeAlerts(c.alerts); err != nil {
				log.Println("failed to store alerts:", err)
			}
		}()
	}()

	alert.Band = band
	if alert.Time.IsZero() {
		for _, b := range c.bandsByDay {
			for _, ba := range b {
				if ba.Name == band {
					alert.Time = ba.Date
					break
				}
			}
		}
	}

	if c.alerts == nil {
		c.alerts = make(map[int64][]*Alert)
		c.alerts[chatID] = []*Alert{alert}
	} else {
		a, ok := c.alerts[chatID]
		if !ok {
			c.alerts[chatID] = []*Alert{alert}
		} else {
			for i, al := range a {
				if al.Band == band {
					a[i] = alert
					return
				}
			}
			c.alerts[chatID] = append(a, alert)
		}
	}
}
