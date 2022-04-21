package main

import (
	"context"

	"go.uber.org/zap"
)

type Message struct {
	ID      uint `gorm:"primaryKey" json:"id"`
	Content string
}

func (s *Server) InstallMessage() {
	var message string
	err := s.conn.QueryRow(context.Background(), "SELECT content from message LIMIT 1").Scan(&message)
	if err != nil {
		s.logger.DPanic("err retrieving message", zap.Error(err))
	}

	s.cachedMessage = []byte(message)
}
