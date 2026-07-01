package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"mrflag/internal/room"
	"mrflag/pkg/protocol"
)

type Handler struct {
	mgr *room.Manager
}

func NewHandler(mgr *room.Manager) *Handler {
	return &Handler{mgr: mgr}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/default-room", h.defaultRoom)
	mux.HandleFunc("/api/rooms", h.rooms)
	mux.HandleFunc("/api/rooms/", h.roomAction)
	mux.HandleFunc("/", h.index)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) defaultRoom(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		defaultRoom, err := h.mgr.EnsureDefaultRoom()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, protocol.RoomInfo{
			RoomID:   defaultRoom.ID,
			JoinCode: defaultRoom.JoinCode,
			Status:   string(defaultRoom.Status),
		})
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (h *Handler) rooms(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/rooms" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, h.mgr.ListRooms())
	case http.MethodPost:
		h.createRoom(w, r)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (h *Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	var cfg protocol.RoomConfig
	if err := decodeJSON(r, &cfg); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "请求 Body 不是合法 JSON", err.Error())
		return
	}
	created, err := h.mgr.CreateRoom(cfg)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, protocol.RoomInfo{
		RoomID:   created.ID,
		JoinCode: created.JoinCode,
		Status:   string(created.Status),
	})
}

func (h *Handler) roomAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/rooms/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	roomID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case action == "" && r.Method == http.MethodGet:
		h.getRoom(w, roomID)
	case action == "start" && r.Method == http.MethodPost:
		h.startRoom(w, roomID)
	case action == "stop" && r.Method == http.MethodPost:
		h.stopRoom(w, r, roomID)
	case action == "scores" && r.Method == http.MethodGet:
		h.scores(w, roomID)
	case action == "flags" && r.Method == http.MethodGet:
		h.flags(w, roomID)
	case action == "scene-map" && r.Method == http.MethodGet:
		h.getSceneMap(w, roomID)
	case action == "scene-map" && r.Method == http.MethodPut:
		h.putSceneMap(w, r, roomID)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) getRoom(w http.ResponseWriter, roomID string) {
	s, err := h.mgr.Snapshot(roomID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) startRoom(w http.ResponseWriter, roomID string) {
	if err := h.mgr.StartGame(roomID); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) stopRoom(w http.ResponseWriter, r *http.Request, roomID string) {
	var payload protocol.StopGamePayload
	if r.Body != nil {
		_ = decodeJSON(r, &payload)
	}
	if err := h.mgr.StopGame(roomID, payload.Reason); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) scores(w http.ResponseWriter, roomID string) {
	scores, err := h.mgr.Scores(roomID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, scores)
}

func (h *Handler) flags(w http.ResponseWriter, roomID string) {
	flags, err := h.mgr.Flags(roomID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flags)
}

func (h *Handler) getSceneMap(w http.ResponseWriter, roomID string) {
	scene, err := h.mgr.SceneMap(roomID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, scene)
}

func (h *Handler) putSceneMap(w http.ResponseWriter, r *http.Request, roomID string) {
	var payload room.SceneMapSavePayload
	if err := decodeJSON(r, &payload); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "请求 Body 不是合法 JSON", err.Error())
		return
	}
	if payload.PlayerID == "" {
		payload.PlayerID = "admin"
	}
	saved, err := h.mgr.SaveSceneMap(roomID, payload)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, dashboardHTML)
}

func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	if ce, ok := room.AsCodedError(err); ok {
		status := http.StatusBadRequest
		if ce.Code == "room_not_found" {
			status = http.StatusNotFound
		}
		writeAPIError(w, status, ce.Code, ce.Message, ce.Detail)
		return
	}
	writeAPIError(w, http.StatusInternalServerError, "internal_error", err.Error(), nil)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string, detail any) {
	writeJSON(w, status, protocol.ErrorPayload{
		Code:    code,
		Message: message,
		Detail:  detail,
	})
}

func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "HTTP 方法不允许", nil)
}
