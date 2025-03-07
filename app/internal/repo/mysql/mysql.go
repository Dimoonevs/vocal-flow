package mysql

import (
	"database/sql"
	"encoding/json"
	"flag"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"log"
	"sync"
)

type Storage struct {
	db *sql.DB
}

var (
	mysqlConnectionString = flag.String("SQLConnPassword", "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4,utf8", "DB connection")
	storage               *Storage
	once                  sync.Once
)

func initMySQLConnection() {
	dbConn, err := sql.Open("mysql", *mysqlConnectionString)
	if err != nil {
		log.Fatal(err)
	}
	dbConn.SetMaxIdleConns(0)

	storage = &Storage{
		db: dbConn,
	}
}

func GetConnection() *Storage {
	once.Do(func() {
		initMySQLConnection()
	})

	return storage
}

func (s *Storage) GetURIByID(ID int) (string, error) {
	query := `
	SELECT filepath FROM files WHERE id = ?
`
	row := s.db.QueryRow(query, ID)
	var path string
	err := row.Scan(&path)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return path, err
}

func (s *Storage) SaveTranscription(subtitlesMap map[string]string, videoID int) error {
	var subtitlesJSON []map[string]string
	for lang, uri := range subtitlesMap {
		subtitlesJSON = append(subtitlesJSON, map[string]string{"lang": lang, "uri": uri})
	}

	subtitlesData, err := json.Marshal(subtitlesJSON)
	if err != nil {
		logrus.Errorf("Failed to marshal subtitles JSON: %v", err)
		return nil
	}

	query := `INSERT INTO video_ai (video_id, subtitles_url) 
	          VALUES (?, ?) 
	          ON DUPLICATE KEY UPDATE subtitles_url = VALUES(subtitles_url)`
	_, err = s.db.Exec(query, videoID, subtitlesData)
	if err != nil {
		logrus.Errorf("Failed to save subtitles to DB: %v", err)
		return nil
	}

	logrus.Infof("Subtitles successfully saved in DB")

	return nil
}
