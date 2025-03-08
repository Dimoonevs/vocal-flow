package mysql

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Dimoonevs/vocal-flow/app/internal/models"
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

func (s *Storage) SaveVideoWithSub(id int, path string) error {
	query := `UPDATE video_ai SET subtitles_video_url = ? WHERE video_id = ?`

	_, err := s.db.Exec(query, path, id)
	if err != nil {
		logrus.Errorf("Failed to save video to DB: %v", err)
		return err
	}
	return nil
}

func (s *Storage) GetVideoSubtitles(id int) ([]models.SubtitlesData, string, string, error) {
	query := `
		SELECT f.filename, f.filepath, v.subtitles_url
		FROM files f
		LEFT JOIN video_ai v ON f.id = v.video_id
		WHERE f.id = ?;
	`

	row := s.db.QueryRow(query, id)
	var filepath, filename string
	var subtitlesURL sql.NullString

	err := row.Scan(&filename, &filepath, &subtitlesURL)
	if err != nil {
		logrus.Errorf("Failed to get subtitles from DB: %v", err)
		return nil, "", "", fmt.Errorf("scan error: %w", err)
	}

	var subtitlesData []models.SubtitlesData

	err = json.Unmarshal([]byte(subtitlesURL.String), &subtitlesData)
	if err != nil {
		logrus.Errorf("Failed to parse subtitles JSON: %w", err)
		return nil, "", "", fmt.Errorf("failed to parse subtitles JSON: %w", err)
	}

	return subtitlesData, filepath, filename, nil
}

func (s *Storage) GetOriginalSubtitles(id int) (string, error) {
	query := `
		SELECT subtitles_url
		FROM video_ai
		WHERE video_id = ?;
`

	row := s.db.QueryRow(query, id)
	var subtitlesURL sql.NullString
	err := row.Scan(&subtitlesURL)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		logrus.Errorf("Failed to get subtitles from DB: %v", err)
		return "", err
	}

	var subtitlesData []models.SubtitlesData
	err = json.Unmarshal([]byte(subtitlesURL.String), &subtitlesData)
	if err != nil {
		logrus.Errorf("Failed to parse subtitles JSON: %w", err)
		return "", err
	}

	for _, subtitles := range subtitlesData {
		if subtitles.Lang == "original" {
			return subtitles.URI, nil
		}
	}
	logrus.Errorf("Failed to get original subtitles from DB: %v", err)
	return "", fmt.Errorf("failed to find original subtitle from DB")
}

func (s *Storage) SaveSummary(summary string, id int) error {
	query := `UPDATE video_ai SET summary = ? WHERE video_id = ?`

	_, err := s.db.Exec(query, summary, id)
	if err != nil {
		logrus.Errorf("Failed to save summary to DB: %v", err)
		return err
	}
	return nil
}
