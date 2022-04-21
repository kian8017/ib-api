package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgtype"
	"go.uber.org/zap"
)

type JsonError struct {
	Error string `json:"error"`
}

func MarshalError(err string) []byte {
	e := JsonError{
		Error: err,
	}

	enc, encErr := json.Marshal(e)
	if encErr != nil {
		panic(encErr)
	}
	return enc
}

// RootHandler is the HTTP handler that handles "/" requests
func (s *Server) RootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("The IndexBrain Server is running."))
}

func (s *Server) LocationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(s.cachedLocations)
}

func (s *Server) SearchHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	analytic := &SearchAnalytic{
		UserId:        uuid.Nil,
		Time:          startTime,
		QueryLocation: pgtype.Int4{Status: pgtype.Null},
	}

	defer func() {
		analytic.Duration = int(time.Since(startTime).Milliseconds())
		s.AddSearchAnalytic(analytic)
	}()

	params := r.URL.Query()
	types, ok := params["type"] // search type
	if !ok || len(types) < 1 {
		analytic.Error = "NO_SEARCH_TYPE"
		w.WriteHeader(http.StatusBadRequest)
		w.Write(MarshalError("no 'type' parameter provided"))
		return
	}

	searchType, ok := NewSearchType(types[0])
	if !ok {
		analytic.Error = "invalid_search_type"
		w.WriteHeader(http.StatusBadRequest)
		w.Write(MarshalError("invalid 'type' parameter provided. Should be one of 'specific', 'fallback', or 'extended'"))
	}

	analytic.Type = searchType

	locations, ok := params["location"]
	if !ok || len(locations) < 1 {
		analytic.Error = "no_location"
		w.WriteHeader(http.StatusBadRequest)
		w.Write(MarshalError("no 'location' parameter provided"))
		return
	}

	entryTypes, ok := params["entry_type"]
	if !ok || len(entryTypes) < 1 {
		analytic.Error = "no_entry_type"
		w.WriteHeader(http.StatusBadRequest)
		w.Write(MarshalError("no 'entry_type' parameter provided"))
		return
	}

	queries, ok := params["query"]
	if !ok || len(queries) < 1 {
		analytic.Error = "no_query"
		w.WriteHeader(http.StatusBadRequest)
		w.Write(MarshalError("no 'query' parameter provided"))
		return
	}

	userIds, ok := params["id"]
	if ok && len(userIds) > 0 {
		analytic.UserId = uuid.FromStringOrNil(userIds[0])
	}

	numRequested := NUM_RESULTS
	numRequesteds, ok := params["num"]
	if ok && len(numRequesteds) > 0 {
		nr, err := strconv.Atoi(numRequesteds[0])
		if err != nil {
			s.logger.Warn("error parsing numRequested", zap.Error(err))
		} else {
			numRequested = nr
		}
	}

	sq, errReason := s.NewSearchQuery(queries[0], locations[0], entryTypes[0])
	if errReason != "" {
		analytic.Error = errReason
		w.WriteHeader(http.StatusBadRequest)
		w.Write(MarshalError(errReason))
		return
	}
	analytic.QueryRaw = sq.Query
	analytic.QueryLocation = pgtype.Int4{Int: int32(sq.Location.ID), Status: pgtype.Present}
	analytic.QueryType = string(sq.Type)

	sq.Query = s.FormatSearch(sq.Query)
	analytic.QueryProcessed = sq.Query

	entries := []Entry{}

	switch searchType {
	case ST_SPECIFIC:
		curEntries, ok := s.IndividualSearch(sq.Query, sq.Location, sq.Type, numRequested)
		if !ok {
			analytic.Error = "invalid_query"
			w.WriteHeader(http.StatusBadRequest)
			w.Write(MarshalError("invalid query"))
			return
		}
		entries = curEntries
	case ST_FALLBACK:
		for _, relId := range sq.Location.RelatedIds {
			if len(entries) >= numRequested {
				break
			}
			curEntries, ok := s.IndividualSearch(sq.Query, s.locations[relId], sq.Type, numRequested-len(entries))
			if !ok {
				analytic.Error = "invalid_query"
				w.WriteHeader(http.StatusBadRequest)
				w.Write(MarshalError("invalid query"))
				return
			}
			entries = append(entries, curEntries...)
		}
	case ST_EXTENDED:
		curEntries, ok := s.ExtendedSearch(sq.Query, sq.Location, sq.Type, numRequested)
		if !ok {
			analytic.Error = "invalid_query"
			w.WriteHeader(http.StatusBadRequest)
			w.Write(MarshalError("invalid query"))
			return
		}
		entries = curEntries
	}

	analytic.NumReturned = len(entries)
	w.Write(MarshalEntries(entries))
	/*
	   	timeBeforeSpecificSearch := time.Now() // Start time
	   	entries, ok := s.IndividualSearch(sq.Query, sq.Location, sq.Type, NUM_RESULTS, req.Logger)
	   	specificSearchTime := time.Since(timeBeforeSpecificSearch) // End time
	   	analytic.AddMeasurement(sq.Location.ID, len(entries), int(specificSearchTime.Milliseconds()))

	   	if !ok {
	   		// Invalid query?
	   		analytic.IsValid = false
	   		req.Write(MT_INVALID_QUERY, RS_INVALID_QUERY)
	   		return
	   	}

	   	results := MarshalEntries(entries, s.logger)
	   	if req.Conn.Canceled.Get() { // Check if canceled
	   		analytic.Cancelled = "before-specific-results"
	   		return
	   	}
	   	req.Write(MT_RESULTS, results)
	   	numReturned += len(entries)

	   	// Fallback
	   	if (numReturned < NUM_RESULTS) && (len(sq.Location.RelatedIds) > 0) {
	   		for _, curLocID := range sq.Location.RelatedIds {
	   			if req.Conn.Canceled.Get() { // Check if canceled
	   				analytic.Cancelled = "before-fallback-search"
	   				return
	   			}
	   			s.logger.Info("fallback search", zap.Int("location_id", curLocID))
	   			avoidExtended = append(avoidExtended, curLocID)

	   			curLoc := s.locations[curLocID]
	   			timeBeforeFallbackSearch := time.Now()
	   			fallbackEntries, ok := s.IndividualSearch(sq.Query, curLoc, sq.Type, (NUM_RESULTS - numReturned), req.Logger)
	   			analytic.AddMeasurement(curLocID, len(fallbackEntries), int(time.Since(timeBeforeFallbackSearch).Milliseconds()))
	   			if !ok {
	   				analytic.IsValid = false
	   				req.Logger.Error("fallback search query not OK", zap.Object(ZAP_SEARCH_QUERY, sq))
	   				req.Write(MT_INVALID_QUERY, RS_INVALID_QUERY)
	   				return
	   			}
	   			results := MarshalEntries(fallbackEntries, s.logger)
	   			if req.Conn.Canceled.Get() { // Check if canceled
	   				analytic.Cancelled = "before-fallback-results"
	   				return
	   			}

	   			req.Write(MT_FALLBACK_RESULTS, results)
	   			numReturned += len(fallbackEntries)
	   		}
	   	}

	   	// Extended
	   	if numReturned < NUM_RESULTS {
	   		if req.Conn.Canceled.Get() { // Check if canceled
	   			analytic.Cancelled = "before-extended-search"
	   			return
	   		}

	   		timeBeforeExtendedSearch := time.Now()
	   		extendedEntries, ok := s.ExtendedSearch(sq.Query, sq.Type, (NUM_RESULTS - numReturned), avoidExtended, req.Logger)
	   		extendedSearchTime := time.Since(timeBeforeExtendedSearch) // End time
	   		analytic.AddMeasurement(-1, len(extendedEntries), int(extendedSearchTime.Milliseconds()))

	   		if !ok {
	   			analytic.IsValid = false
	   			req.Logger.Error("extended search query not OK", zap.Object(ZAP_SEARCH_QUERY, sq))
	   			req.Write(MT_INVALID_QUERY, RS_INVALID_QUERY)
	   			return
	   		} else {
	   			extendedResults := MarshalEntries(extendedEntries, s.logger)
	   			if req.Conn.Canceled.Get() { // Check if canceled
	   				analytic.Cancelled = "before-extended-results"
	   				return
	   			}
	   			req.Write(MT_EXTENDED_RESULTS, extendedResults)
	   		}
	   	}
	   	analytic.IsFinished = true
	   }
	*/
}

func (s *Server) CountsHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	locations, ok := params["location"]
	if !ok || len(locations) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(MarshalError("no 'location' parameter provided"))
		return
	}

	entryTypes, ok := params["entry_type"]
	if !ok || len(entryTypes) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(MarshalError("no 'entry_type' parameter provided"))
		return
	}

	location, ok := s.LookupLocationByAbbr(locations[0])
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(MarshalError("invalid location"))
	}

	entryType, ok := NewEntryType(entryTypes[0])
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(MarshalError("invalid type"))
	}

	var fallbackCharacters int64

	for _, relID := range location.RelatedIds {
		fallbackCharacters += s.fileLengths[entryType][relID]
	}

	enc, err := json.Marshal(Counts{
		Specific: s.fileLengths[entryType][location.ID],
		Fallback: fallbackCharacters,
		Extended: s.totalLengths[entryType],
	})

	if err != nil {
		s.logger.DPanic("error encoding counts", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(MarshalError("internal error"))
	}

	w.Write(enc)
}

func (s *Server) MessageHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(s.cachedMessage)
}

func (s *Server) CouldBesHandler(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	queries, ok := params["query"]
	if !ok || len(queries) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(MarshalError("no 'query' parameter provided"))
		return
	}
	query := queries[0]

	cb := []CouldBe{}

	for k, v := range s.cachedCouldBes {
		if strings.Contains(query, k) {
			cb = append(cb, CouldBe{Key: k, Val: v})
		}
	}

	enc, err := json.Marshal(cb)
	if err != nil {
		s.logger.DPanic("error marshaling couldbes", zap.Error(err))
	}

	w.Write(enc)
}

func (s *Server) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	s.Refresh()
	w.Write([]byte("Refreshed"))
}
