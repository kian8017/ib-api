package main

import (
	"encoding/json"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Message Types

const (
	// General
	MT_SUCCESS string = "success"
	MT_FAILURE string = "failure"

	// Replacements
	MT_GET_REPLACEMENTS string = "get_replacements"

	// CouldBes
	MT_GET_COULD_BES string = "get_could_bes"

	// Locations
	MT_GET_LOCATIONS string = "get_locations"

	// Message
	MT_GET_MESSAGE string = "get_message"

	// Search
	MT_SEARCH           string = "search"
	MT_CANCEL_SEARCH    string = "cancel_search"
	MT_RESULTS          string = "results"
	MT_FALLBACK_RESULTS string = "fallback_results"
	MT_EXTENDED_RESULTS string = "extended_results"

	MT_QUERY_MESSAGE  string = "query_message"
	MT_SPECIFIC_COUNT string = "count"
	MT_EXTENDED_COUNT string = "extended_count"
	MT_INVALID_QUERY  string = "invalid_query"
	// Reasons for invalid query:
	RS_INVALID_JSON     string = "invalid_json"
	RS_INVALID_TYPE     string = "invalid_type"
	RS_INVALID_LOCATION string = "invalid_location"
	RS_INVALID_QUERY    string = "invalid_query"
	RS_BLANK_QUERY      string = "blank_query"
)

// Message represents a single message.
type BaseMessage struct {
	Type    string `json:"type"`
	Data    string `json:"data"`
	Channel int    `json:"channel"` // Used for associating request / responses
}

// NewMessage creates a message from a Type, Data, and Channel.
func NewBaseMessage(typ string, data string, channel int) BaseMessage {
	return BaseMessage{Type: typ, Data: data, Channel: channel}
}

// UnmarshalMessage creates a message from a JSON byte slice.
func UnmarshalMessage(m []byte, logger *zap.Logger) (BaseMessage, bool) {
	a := BaseMessage{}
	err := json.Unmarshal(m, &a)
	if err != nil {
		logger.Error("error decoding", zap.Error(err), zap.String(ZAP_MESSAGE_RAW, string(m)))
		return a, false
	}

	if a.Type == "" {
		logger.Warn("no type specified, invalid message", zap.Object(ZAP_MESSAGE, a))
		return a, false
	}
	return a, true
}

// MarshalMessage encodes a message into JSON.
func MarshalMessage(m BaseMessage, logger *zap.Logger) []byte {
	ser, err := json.Marshal(m)
	if err != nil {
		logger.DPanic("error encoding", zap.Error(err), zap.Object(ZAP_MESSAGE, m))
	}
	return ser
}

// MarshalLogObject allows Messages to be logged with Zap.
func (m BaseMessage) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("type", m.Type)
	enc.AddString("data", m.Data)
	enc.AddInt("channel", m.Channel)
	return nil
}
