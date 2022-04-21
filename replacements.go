package main

import (
	"context"
	"strings"

	"go.uber.org/zap"
)

// Replacement represents a custom substitution in a query, used to make regular expressions more indexing-friendly.
type Replacement struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

// InstallReplacements retrieves all replacements from the database and populates the server's cache.
func (s *Server) InstallReplacements() {
	s.cachedReplacements = make(map[string]string)

	rows, _ := s.conn.Query(context.Background(), "SELECT key, val FROM replacements")
	defer rows.Close()
	for rows.Next() {
		var key, val string
		err := rows.Scan(&key, &val)
		if err != nil {
			s.logger.DPanic("error getting replacement", zap.Error(err))
			continue
		}
		s.cachedReplacements[key] = val
	}

	s.logger.Info("replacements", zap.Int("num", len(s.cachedReplacements)))
}

// DoReplacements takes the query, substitutes any replacements, and then returns the final query.
func (s *Server) DoReplacements(q string) string {
	for k, v := range s.cachedReplacements {
		if strings.Contains(q, k) {
			q = strings.ReplaceAll(q, k, v)
		}
	}
	return q
}
