package main

import (
	"context"
	"encoding/json"
	"os"
	"sort"

	"go.uber.org/zap"
)

// InstallLocations grabs the list of locations from the nameFolder and populates the server's cache.
func (s *Server) InstallLocations() {
	dirEntries, err := os.ReadDir(NAME_FOLDER)
	if err != nil {
		s.logger.Panic(err.Error())
	}
	d := []Location{}
	for _, de := range dirEntries {
		if de.IsDir() {
			if de.Name() == ".stfolder" {
				continue
			}
			nc, ok := NewLocation(de.Name())
			if ok {
				d = append(d, nc)
			} else {
				s.logger.Warn("NewLocation length was not 2, ignoring", zap.String(ZAP_LOCATION, de.Name()))
			}
		}
	}

	numUpdated := 0
	s.locations = make(map[int]*Location)

	rows, _ := s.conn.Query(context.Background(), "SELECT id, abbr, name, is_language FROM locations")
	// Go through rows we already have
	for rows.Next() {
		var id int
		var abbr, name string
		var is_language bool

		err = rows.Scan(&id, &abbr, &name, &is_language)
		if err != nil {
			s.logger.Error("error reading location", zap.Error(err))
			continue
		}

		s.locations[id] = &Location{
			ID:         id,
			Abbr:       abbr,
			Name:       name,
			IsLanguage: is_language,
		}
	}

	// And now check to make sure we aren't missing any
	for _, curFSLocation := range d {
		ok := false
		for _, curDBLocation := range s.locations {
			if curFSLocation.Abbr == curDBLocation.Abbr {
				ok = true
				break
			}
		}

		if ok {
			// We already have that entry
			continue
		}

		// If we don't, we need to add it to the database
		var id int
		err := s.conn.QueryRow(context.Background(), "INSERT INTO locations (abbr, name, is_language) VALUES($1, $2, $3) RETURNING id", curFSLocation.Abbr, curFSLocation.Name, false).Scan(&id)
		if err != nil {
			s.logger.Error("error adding location", zap.Error(err))
			continue
		}
		numUpdated++
		s.locations[id] = &Location{
			ID:         id,
			Abbr:       curFSLocation.Abbr,
			Name:       curFSLocation.Name,
			IsLanguage: false,
		}
		s.logger.Info("added location to the database", zap.String("abbr", curFSLocation.Abbr))
	}

	// Sorting this way means we should be able to just append results to the relatedIDs for each location
	rows, _ = s.conn.Query(context.Background(), "SELECT location_id, related_id FROM related_locations ORDER BY location_id, sort")
	for rows.Next() {
		var locationID, relatedID int
		err := rows.Scan(&locationID, &relatedID)
		if err != nil {
			s.logger.Error("error reading related_locations", zap.Error(err))
			continue
		}

		s.locations[locationID].RelatedIds = append(s.locations[locationID].RelatedIds, relatedID)
		// s.logger.Info("adding related connection", zap.Int("location_id", locationID), zap.Int("related_id", relatedID))
	}

	s.logger.Info("updated locations", zap.Int("num_updated", numUpdated))
	s.CacheLocations()
}

func (s *Server) CacheLocations() {
	var loc []*Location

	for _, l := range s.locations {
		loc = append(loc, l)
	}

	sort.Slice(loc, func(i, j int) bool {
		return loc[i].Abbr < loc[j].Abbr
	})
	enc, err := json.Marshal(loc)
	if err != nil {
		s.logger.DPanic("error marshaling locations", zap.Error(err))
		return
	}
	s.cachedLocations = enc
}

// LookupLocationByAbbr takes a locationAbbr and returns the associated Location.
func (s *Server) LookupLocationByAbbr(locationAbbr string) (*Location, bool) {
	for _, e := range s.locations {
		if e.Abbr == locationAbbr {
			return e, true
		}
	}
	return &Location{}, false
}
