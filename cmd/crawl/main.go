package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/anthrotech-dev/activity"
)

type Userdata struct {
	Name    string `json:"name"`
	GitHub  string `json:"github"`
	Discord int64  `json:"discord"`
}

type Members map[string]Userdata

func main() {

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://anthrotech.dev/members.json")
	if err != nil {
		log.Fatalf("Error fetching members.json: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error fetching members.json: status code %d", resp.StatusCode)
	}

	var members Members
	err = json.NewDecoder(resp.Body).Decode(&members)
	if err != nil {
		log.Fatalf("Error decoding members.json: %v", err)
	}

	dsn := os.Getenv("DB_DSN")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             300 * time.Millisecond, // Slow SQL threshold
			LogLevel:                  logger.Warn,            // Log level
			IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,                   // Enable color
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:         gormLogger,
		TranslateError: true,
	})
	if err != nil {
		panic("failed to connect database")
	}

	err = db.WithContext(ctx).AutoMigrate(&activity.Activity{})
	if err != nil {
		log.Fatalf("Error during AutoMigrate: %v", err)
	}

	acts := []activity.Activity{}

	progressActs, err := collectDiscordProgress()
	if err != nil {
		log.Fatalf("Error collecting Discord activities: %v", err)
	}
	for _, act := range progressActs {
		var anthrotech_id string
		for _, u := range members {
			if act.UserID == fmt.Sprintf("%d", u.Discord) {
				anthrotech_id = u.GitHub
				break
			}
		}

		if anthrotech_id != "" {
			act.UserID = anthrotech_id
			acts = append(acts, act)
		}
	}

	joinActs, err := collectJoin()
	if err != nil {
		log.Fatalf("Error collecting Discord join activities: %v", err)
	}
	for _, act := range joinActs {
		var anthrotech_id string
		for _, u := range members {
			if act.UserID == fmt.Sprintf("%d", u.Discord) {
				anthrotech_id = u.GitHub
				break
			}
		}
		if anthrotech_id != "" {
			act.UserID = anthrotech_id
			acts = append(acts, act)
		}
	}

	githubActs, err := collectGitHub(ctx)
	if err != nil {
		log.Fatalf("Error collecting GitHub activities: %v", err)
	}
	for _, act := range githubActs {
		var anthrotech_id string
		for _, u := range members {
			if act.UserID == u.GitHub {
				anthrotech_id = u.GitHub
				break
			}
		}
		if anthrotech_id != "" {
			act.UserID = anthrotech_id
			acts = append(acts, act)
		}
	}

	data, err := json.MarshalIndent(acts, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling activities to JSON: %v", err)
	}

	log.Println(string(data))

	for _, act := range acts {
		err := db.Save(&act).Error
		if err != nil {
			log.Printf("Error inserting activity ID %s: %v", act.ID, err)
		}
	}
}
