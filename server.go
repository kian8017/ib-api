package main

import (
	"context"
	"net/http"

	"github.com/jackc/pgtype"
	pgtypeuuid "github.com/jackc/pgtype/ext/gofrs-uuid"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/cors"

	"go.uber.org/zap"
)

const (
	NAME_FOLDER = "/names"
	LISTEN_ADDR = ":80"
)

type Server struct {
	// env variable
	dbString string

	logger *zap.Logger

	httpHandler http.Handler

	conn *pgxpool.Pool

	locations map[int]*Location // id -> location

	// This is what is cached and needs to be refreshed when updated through Directus
	cachedLocations    []byte            // JSON encoded list of locations, in order to avoid having to reparse again and again
	cachedReplacements map[string]string // key -> val
	cachedCouldBes     map[string]string // key -> val
	cachedMessage      []byte

	// For counts
	fileLengths  map[EntryType]map[int]int64 // fileLengths[EntryType][Location.Id] = length
	totalLengths map[EntryType]int64
}

// NewServer creates a new Server.
func NewServer(dbString string, logger *zap.Logger) *Server {
	s := Server{dbString: dbString, logger: logger}

	s.InstallDB()
	s.InstallHTTP()
	s.Refresh() // Install refreshable things
	return &s
}

func (s *Server) Refresh() {
	s.logger.Info("starting refresh")
	s.InstallLocations()
	s.InstallFileLengths() // Needs to be after InstallLocations
	s.InstallReplacements()
	s.InstallCouldBes()
	s.InstallMessage()
	s.logger.Info("refreshed")
}

// Run starts the Server.
func (s *Server) Run() {
	s.logger.Fatal("server error", zap.Error(http.ListenAndServe(LISTEN_ADDR, s.httpHandler)))
}

func (s *Server) InstallHTTP() {
	// Set up various portions of the server
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.RootHandler)
	mux.HandleFunc("/locations", s.LocationsHandler)
	mux.HandleFunc("/search", s.SearchHandler)
	mux.HandleFunc("/counts", s.CountsHandler)
	mux.HandleFunc("/message", s.MessageHandler)
	mux.HandleFunc("/couldbes", s.CouldBesHandler)
	mux.HandleFunc("/refresh", s.RefreshHandler)
	c := cors.AllowAll()

	s.httpHandler = c.Handler(mux)
}

func (s *Server) InstallDB() {
	dbConfig, err := pgxpool.ParseConfig(s.dbString)
	if err != nil {
		s.logger.Panic("error creating db config", zap.Error(err))
	}

	dbConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		conn.ConnInfo().RegisterDataType(pgtype.DataType{
			Value: &pgtypeuuid.UUID{},
			Name:  "uuid",
			OID:   pgtype.UUIDOID,
		})
		return nil
	}

	conn, err := pgxpool.Connect(context.Background(), s.dbString)
	if err != nil {
		s.logger.Panic("Couldn't connect to database")
	}

	err = conn.Ping(context.Background())
	if err != nil {
		s.logger.Panic("error pinging database", zap.Error(err))
	}

	_, err = conn.Exec(context.Background(), SQL_CREATE_TABLES)

	if err != nil {
		s.logger.Panic("error creating tables", zap.Error(err))
	}

	s.conn = conn
}
