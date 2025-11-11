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

type member struct {
	User struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		GlobalName string `json:"global_name"`
	} `json:"user"`
	JoinedAt time.Time `json:"joined_at"` // RFC3339/3339Nano を自動デコード
}

func collectJoin() ([]activity.Activity, error) {

	token := os.Getenv("DISCORD_BOT_TOKEN")
	guildID := "1378710940807073913"

	client := &http.Client{Timeout: 30 * time.Second}

	var (
		after         = "0" // 最初は0（これより大きいIDが返る）
		lastBatchSize = 0
	)

	result := []activity.Activity{}

	for {
		url := fmt.Sprintf("%s/guilds/%s/members?limit=%d&after=%s", baseURL, guildID, maxPerRequest, after)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bot "+token)
		req.Header.Set("User-Agent", defaultUserAgent)

		res, err := client.Do(req)
		if err != nil {
			log.Fatalf("request error: %v", err)
		}

		// 429 レート制限
		if res.StatusCode == http.StatusTooManyRequests {
			retryAfter := res.Header.Get("Retry-After")
			_ = res.Body.Close()
			sleep := time.Second
			if retryAfter != "" {
				if ms, err := strconv.ParseFloat(retryAfter, 64); err == nil {
					// 秒かミリ秒か曖昧なため判定
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

		var members []member
		if err := json.NewDecoder(res.Body).Decode(&members); err != nil {
			_ = res.Body.Close()
			log.Fatalf("decode error: %v", err)
		}
		_ = res.Body.Close()

		lastBatchSize = len(members)
		if lastBatchSize == 0 {
			break
		}

		// 書き出し & 次ページ用 after を更新（最大IDを使う）
		maxID := after

		for _, m := range members {
			joinedUTC := m.JoinedAt.UTC()

			act := activity.Activity{
				ID:        "join-" + m.User.ID,
				UserID:    m.User.ID,
				Timestamp: joinedUTC,
				Type:      "join",
				IsSpecial: true,
			}

			result = append(result, act)

			if m.User.ID > maxID {
				maxID = m.User.ID
			}
		}

		after = maxID
		if lastBatchSize < maxPerRequest {
			break // 取り切り
		}
	}

	return result, nil

}
