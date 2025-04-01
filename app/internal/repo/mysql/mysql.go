package mysql

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	libVideo "github.com/Dimoonevs/video-service/app/pkg/lib"
	"github.com/Dimoonevs/vocal-flow/app/internal/lib"
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
			return libVideo.GetVideoLocalLink(subtitles.URI), nil
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

func (s *Storage) GetDataByVideoID(id int) (*models.DataAI, error) {
	query := `SELECT 
		video_id, 
		subtitles_url, 
		subtitles_video_url, 
		translate_video_url, 
		summary
	FROM video_ai 
	WHERE video_id = ?;`

	row := s.db.QueryRow(query, id)
	dataAI := &models.DataAI{}
	err := row.Scan(&dataAI.VideoId, &dataAI.SubtitlesURL, &dataAI.SubtitlesVideoURL, &dataAI.TranslateVideoURL, &dataAI.Summary)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		logrus.Errorf("Failed to get subtitles from DB: %v", err)
		return nil, err
	}

	lib.TransformDataAI(dataAI)
	return dataAI, nil
}

func (s *Storage) GetAllDataByUserID(userID int) ([]*models.DataAI, error) {
	query := `SELECT 
		va.video_id, 
		va.subtitles_url, 
		va.subtitles_video_url, 
		va.translate_video_url, 
		va.summary
	FROM video_ai va
	JOIN files f ON va.video_id = f.id
	WHERE f.user_id = ?;`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		logrus.Errorf("Failed to query data by user_id: %v", err)
		return nil, err
	}
	defer rows.Close()

	result := []*models.DataAI{}

	for rows.Next() {
		dataAI := &models.DataAI{}
		err := rows.Scan(
			&dataAI.VideoId,
			&dataAI.SubtitlesURL,
			&dataAI.SubtitlesVideoURL,
			&dataAI.TranslateVideoURL,
			&dataAI.Summary,
		)
		if err != nil {
			logrus.Errorf("Failed to scan row: %v", err)
			continue
		}

		lib.TransformDataAI(dataAI)
		result = append(result, dataAI)
	}

	if err = rows.Err(); err != nil {
		logrus.Errorf("Rows iteration error: %v", err)
		return nil, err
	}

	return result, nil
}

func (s *Storage) GetUserSetting(userID, settingID int) (*models.UserSettings, error) {
	query := `SELECT id, user_id, token, gpt_model, whisper_model, tts_model, name 
	          FROM user_ai_settings WHERE user_id = ? AND id = ?`

	var settings models.UserSettings
	row := s.db.QueryRow(query, userID, settingID)

	if err := row.Scan(&settings.ID, &settings.UserID, &settings.AIToken, &settings.GPTModel, &settings.WhisperModel, &settings.TTSModel, &settings.Name); err != nil {
		if err == sql.ErrNoRows {
			logrus.Infof("User settings not found by setting ")
			return nil, fmt.Errorf("user settings not found")
		}
		logrus.Errorf("Cannot get user setting: %v", err)
		return nil, err
	}

	return &settings, nil
}
