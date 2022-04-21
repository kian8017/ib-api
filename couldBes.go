package main

import (
	"context"

	"go.uber.org/zap"
)

// CouldBe represents a FIXME
type CouldBe struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

func (s *Server) InstallCouldBes() {
	s.cachedCouldBes = make(map[string]string)

	rows, _ := s.conn.Query(context.Background(), "SELECT key, val FROM could_bes")
	defer rows.Close()
	for rows.Next() {
		var key, val string
		err := rows.Scan(&key, &val)
		if err != nil {
			s.logger.DPanic("error getting couldBe", zap.Error(err))
			continue
		}
		s.cachedCouldBes[key] = val
	}
	s.logger.Info("couldBes", zap.Int("num", len(s.cachedCouldBes)))

}
