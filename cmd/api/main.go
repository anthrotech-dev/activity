package main

import (
	"log"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/anthrotech-dev/activity"
)

type DailyActivity struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
}

func main() {
	dsn := os.Getenv("DB_DSN")

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

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/:id", func(c echo.Context) error {

		id := c.Param("id")

		since := time.Now().AddDate(-1, 0, 0)
		until := time.Now()

		sinceQ := c.QueryParam("since")
		if sinceQ != "" {
			parsedSince, err := time.Parse("2006-01-02", sinceQ)
			if err == nil {
				since = parsedSince
			}
		}
		untilQ := c.QueryParam("until")
		if untilQ != "" {
			parsedUntil, err := time.Parse("2006-01-02", untilQ)
			if err == nil {
				until = parsedUntil
			}
		}

		var results []DailyActivity
		db.
			Model(&activity.Activity{}).
			Select("DATE(timestamp) as date, COUNT(*) as count").
			Where("user_id = ? AND timestamp >= ? AND timestamp <= ?", id, since, until).
			Group("date").
			Order("date").
			Scan(&results)

		resultsMap := make(map[string]int)
		for _, r := range results {
			resultsMap[r.Date.Format("2006-01-02")] = r.Count
		}

		return c.JSON(200, resultsMap)
	})

	e.GET("/:id/total", func(c echo.Context) error {
		id := c.Param("id")
		var count int64
		db.
			Model(&activity.Activity{}).
			Where("user_id = ?", id).
			Count(&count)

		return c.JSON(200, map[string]int64{"total": count})
	})

	e.GET("/:id/:date", func(c echo.Context) error {
		id := c.Param("id")
		dateStr := c.Param("date")
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return c.JSON(400, map[string]string{"error": "invalid date format"})
		}
		var activities []activity.Activity
		db.
			Where("user_id = ? AND DATE(timestamp) = ?", id, date).
			Find(&activities)
		return c.JSON(200, activities)
	})

	e.Logger.Fatal(e.Start(":3000"))
}
