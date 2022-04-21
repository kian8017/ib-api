package main

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgtype"
	"go.uber.org/zap"
)

type SearchAnalytic struct {
	UserId         uuid.UUID
	Type           SearchType
	Time           time.Time
	Duration       int
	NumReturned    int
	Error          string
	QueryRaw       string
	QueryProcessed string
	QueryLocation  pgtype.Int4
	QueryType      string
}

func (s *Server) AddSearchAnalytic(sa *SearchAnalytic) {
	_, err := s.conn.Exec(context.Background(), "INSERT INTO data_searches (user_id, search_type, time, duration, num_returned, error, query_raw, query_processed, query_location, query_type) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
		sa.UserId, sa.Type, sa.Time, sa.Duration, sa.NumReturned, sa.Error, sa.QueryRaw, sa.QueryProcessed, sa.QueryLocation, sa.QueryType)
	if err != nil {
		s.logger.Error("error inserting into data_searches", zap.Error(err))
		return
	}
}
