package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/anthrotech-dev/activity"
)

const (
	baseURL          = "https://discord.com/api/v10"
	discordEpochMs   = int64(1420070400000) // 2015-01-01T00:00:00Z
	maxPerRequest    = 100
	defaultUserAgent = "Anthrotech Bot"
)

type message struct {
	ID        string    `json:"id"`
	Type      int       `json:"type"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"` // RFC3339
	Author    struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"author"`
}

func snowflakeForTimeUTC(t time.Time) string {
	ms := t.UTC().UnixMilli()
	sf := (ms - discordEpochMs) << 22
	return strconv.FormatInt(sf, 10)
}

func collectDiscord() ([]activity.Activity, error) {

	token := os.Getenv("DISCORD_BOT_TOKEN")
	channelID := "1381317464805085204"

	jst, _ := time.LoadLocation("Asia/Tokyo")
	nowJST := time.Now().In(jst)

	startJST := time.Date(nowJST.Year(), nowJST.Month(), nowJST.Day()-2, 0, 0, 0, 0, jst)
	endJST := time.Date(nowJST.Year(), nowJST.Month(), nowJST.Day()-1, 23, 59, 59, 999000000, jst)

	startUTC, endUTC := startJST.UTC(), endJST.UTC()
	endSF := snowflakeForTimeUTC(endUTC)

	client := &http.Client{Timeout: 30 * time.Second}

	result := []activity.Activity{}

FETCH_LOOP:
	for {
		url := fmt.Sprintf("%s/channels/%s/messages?limit=%d&before=%s", baseURL, channelID, maxPerRequest, endSF)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bot "+token)
		req.Header.Set("User-Agent", defaultUserAgent)

		res, err := client.Do(req)
		if err != nil {
			log.Fatalf("request error: %v", err)
		}

		if res.StatusCode == http.StatusTooManyRequests {
			retryAfter := res.Header.Get("Retry-After")
			_ = res.Body.Close()
			sleep := time.Second
			if retryAfter != "" {
				if ms, err := strconv.ParseFloat(retryAfter, 64); err == nil {
					if ms > 1000 {
						sleep = time.Duration(ms) * time.Millisecond
					} else {
						sleep = time.Duration(ms*1000) * time.Millisecond
					}
				}
			}
			time.Sleep(sleep)
			continue
		}

		if res.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(res.Body)
			_ = res.Body.Close()
			log.Fatalf("HTTP %d: %s", res.StatusCode, string(body))
		}

		var msgs []message
		if err := json.NewDecoder(res.Body).Decode(&msgs); err != nil {
			_ = res.Body.Close()
			log.Fatalf("decode error: %v", err)
		}
		_ = res.Body.Close()

		if len(msgs) == 0 {
			break
		}

		for _, m := range msgs {
			t := m.Timestamp.UTC()
			if t.Before(startUTC) {
				break FETCH_LOOP
			}
			if m.Type != 0 {
				continue
			}
			fmt.Printf("[Discord] %s %s %s(%s): %s\n",
				m.ID,
				m.Timestamp.In(jst).Format("2006-01-02 15:04:05 MST"),
				m.Author.ID, m.Author.Username, m.Content)

			act := activity.Activity{
				ID:        "discord-" + m.ID,
				UserID:    m.Author.ID,
				Timestamp: m.Timestamp,
				Type:      "common",
				Body:      m.Content,
			}

			result = append(result, act)
		}

		if len(msgs) < maxPerRequest {
			break FETCH_LOOP
		}

		endSF = msgs[len(msgs)-1].ID
	}

	return result, nil
}
