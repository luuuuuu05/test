package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mrflag/internal/room"
	"mrflag/pkg/protocol"
)

func TestRoomsGETListsSummaries(t *testing.T) {
	mgr := room.NewManager(room.ManagerConfig{})
	created, err := mgr.CreateRoom(protocol.RoomConfig{RoomName: "alpha"})
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	if _, _, err := mgr.JoinRoom(created.JoinCode, "P_001", "A", "Quest", nil); err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(mgr).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var rooms []room.RoomSummary
	if err := json.NewDecoder(rec.Body).Decode(&rooms); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(rooms) != 1 {
		t.Fatalf("len(rooms) = %d, want 1", len(rooms))
	}
	if rooms[0].RoomID != created.ID || rooms[0].PlayerCount != 1 {
		t.Fatalf("room summary = %#v, want id=%s player_count=1", rooms[0], created.ID)
	}
}

func TestIndexServesDashboard(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(room.NewManager(room.ManagerConfig{})).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(rec.Body.String(), "MR Flag 管理端") {
		t.Fatal("dashboard title missing")
	}
}

func TestDefaultRoomEndpointEnsuresRoom(t *testing.T) {
	mgr := room.NewManager(room.ManagerConfig{})
	mux := http.NewServeMux()
	NewHandler(mgr).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/default-room", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var info protocol.RoomInfo
	if err := json.NewDecoder(rec.Body).Decode(&info); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if info.RoomID != "ROOM_DEFAULT" || info.JoinCode != "DEFAULT" {
		t.Fatalf("default room info = %#v", info)
	}
}
