package route

import (
	"encoding/json"
	"fmt"
	"github.com/Dimoonevs/user-service/app/pkg/jwt"
	"github.com/Dimoonevs/video-service/app/pkg/respJSON"
	"github.com/Dimoonevs/vocal-flow/app/internal/models"
	"github.com/Dimoonevs/vocal-flow/app/internal/repo/mysql"
	"github.com/Dimoonevs/vocal-flow/app/internal/service"
	"github.com/valyala/fasthttp"
	"strings"
)

func RequestHandler(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) == fasthttp.MethodOptions {
		ctx.SetStatusCode(fasthttp.StatusOK)
		return
	}

	path := string(ctx.URI().Path())

	if !strings.HasPrefix(path, "/api-ai") {
		respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, nil, "Endpoint not found")
		return
	}

	jwt.JWTMiddleware(func(ctx *fasthttp.RequestCtx) {
		handleRoutes(ctx, path)
	})(ctx)

}

func handleRoutes(ctx *fasthttp.RequestCtx, path string) {
	remainingPath := path[len("/api-ai"):]

	switch {
	case remainingPath == "/transcription" && ctx.IsPost():
		handleTranscriptionVideo(ctx)
	case remainingPath == "/stitching/subtitles" && ctx.IsPost():
		handleStitchingSub(ctx)
	case remainingPath == "/summary" && ctx.IsPost():
		handleSummaryVideo(ctx)
	case remainingPath == "" && ctx.IsGet():
		handleGetData(ctx)
	case remainingPath == "/translate":
		handleTranslateVideo(ctx)
	default:
		respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, nil, "Endpoint not found")
	}
}

func handleTranscriptionVideo(ctx *fasthttp.RequestCtx) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusUnauthorized, err, "Error getting user id: ")
		return
	}
	body := ctx.PostBody()

	var req models.TranscriptionRequest

	if err := json.Unmarshal(body, &req); err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, err, "Invalid JSON body")
		return
	}

	if req.ID == 0 {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, nil, "ID video not specified or invalid")
		return
	}
	if req.SettingID == 0 {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, nil, "SettingID not specified or invalid")
		return
	}

	if len(req.Langs) == 0 {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, nil, "No languages provided in 'len'")
		return
	}
	go service.CreateTranscription(req.ID, req.Langs, userID, req.SettingID)

	respJSON.WriteJSONResponse(ctx, fasthttp.StatusCreated, "Create transcription in process", nil)
}

func handleStitchingSub(ctx *fasthttp.RequestCtx) {
	id := ctx.QueryArgs().GetUintOrZero("id")

	if id == 0 {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, nil, "ID video not specified or invalid")
		return
	}

	path, err := service.StitchSubtitlesIntoVideo(id)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, err, "Failed to stitch subtitles")
		return
	}
	respJSON.WriteJSONResponse(ctx, fasthttp.StatusCreated, "Created stitch subtitles into video", path)
}

func handleTranslateVideo(ctx *fasthttp.RequestCtx) {

}

func handleSummaryVideo(ctx *fasthttp.RequestCtx) {
	body := ctx.PostBody()
	var req models.TranscriptionRequest

	if err := json.Unmarshal(body, &req); err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, err, "Invalid JSON body")
		return
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusUnauthorized, err, "Error getting user id: ")
		return
	}
	if req.SettingID == 0 || len(req.Langs) == 0 {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, nil, "SettingID or Lang not specified or invalid")
		return
	}
	go service.GetSummary(req.ID, userID, req.SettingID, req.Langs[0])

	respJSON.WriteJSONResponse(ctx, fasthttp.StatusCreated, "Create summary in process", nil)
}

func handleGetData(ctx *fasthttp.RequestCtx) {
	id := ctx.QueryArgs().GetUintOrZero("id")
	if id == 0 {
		userID, err := getUserIDFromContext(ctx)
		if err != nil {
			respJSON.WriteJSONError(ctx, fasthttp.StatusUnauthorized, err, "Error getting user id: ")
			return
		}

		response, err := mysql.GetConnection().GetAllDataByUserID(userID)
		if err != nil {
			respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, err, "Failed to get data")
			return
		}
		respJSON.WriteJSONResponse(ctx, fasthttp.StatusCreated, "Data AI", response)
		return
	}

	response, err := mysql.GetConnection().GetDataByVideoID(id)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, err, "Failed to get data")
		return
	}
	respJSON.WriteJSONResponse(ctx, fasthttp.StatusCreated, "Data AI", response)
}

func getUserIDFromContext(ctx *fasthttp.RequestCtx) (int, error) {
	userIDValue := ctx.UserValue("userID")
	userIDFloat, ok := userIDValue.(float64)
	if !ok {
		return 0, fmt.Errorf("invalid userID format: %f", userIDFloat)
	}

	return int(userIDFloat), nil
}
