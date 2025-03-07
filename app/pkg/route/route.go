package route

import (
	"encoding/json"
	"github.com/Dimoonevs/video-service/app/pkg/respJSON"
	"github.com/Dimoonevs/vocal-flow/app/internal/models"
	"github.com/Dimoonevs/vocal-flow/app/internal/service"
	"github.com/valyala/fasthttp"
	"strings"
)

func RequestHandler(ctx *fasthttp.RequestCtx) {
	path := string(ctx.URI().Path())

	if !strings.HasPrefix(path, "/api-ai") {
		respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, nil, "Endpoint not found")
		return
	}

	remainingPath := path[len("/api-ai"):]

	switch remainingPath {
	case "/transcription":
		handleTranscriptionVideo(ctx)
	case "/stitching/subtitles":
		handleStitchingSub(ctx)
	case "/summary":
		//handleSummaryVideo(ctx)
	case "/translate":
		handleTranslateVideo(ctx)
	default:
		respJSON.WriteJSONError(ctx, fasthttp.StatusNotFound, nil, "Endpoint not found")
	}

}

func handleTranscriptionVideo(ctx *fasthttp.RequestCtx) {
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

	if len(req.Langs) == 0 {
		respJSON.WriteJSONError(ctx, fasthttp.StatusBadRequest, nil, "No languages provided in 'len'")
		return
	}
	resp, err := service.CreateTranscription(req.ID, req.Langs)
	if err != nil {
		respJSON.WriteJSONError(ctx, fasthttp.StatusInternalServerError, err, "Failed to create transcription")
		return
	}
	respJSON.WriteJSONResponse(ctx, fasthttp.StatusCreated, "Created transcription", resp)
}

func handleStitchingSub(ctx *fasthttp.RequestCtx) {

}

func handleTranslateVideo(ctx *fasthttp.RequestCtx) {

}
