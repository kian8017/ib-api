package main

import (
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any

	godotenv.Load()

	// Environmental variables
	dbString := os.Getenv("DB_STRING")
	if dbString == "" {
		logger.Fatal("no DB_STRING provided")
	}

	s := NewServer(dbString, logger)

	s.Run()
}
