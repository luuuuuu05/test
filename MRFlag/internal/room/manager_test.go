package room

import (
	"encoding/json"
	"testing"

	"mrflag/pkg/protocol"
)

func TestSaveSceneMapVersionConflict(t *testing.T) {
	mgr := NewManager(ManagerConfig{
		DefaultDuration:  180,
		DefaultMaxFlags:  5,
		DefaultMinFlags:  4,
		RespawnDelay:     2,
		GrabDistance:     1.5,
		DoubleDuration:   20,
		MaxDoubleItems:   1,
		SceneMapMaxBytes: 1024,
	})
	r, err := mgr.CreateRoom(protocol.RoomConfig{
		RoomName: "test",
		Bounds:   protocol.Bounds{XMin: -5, XMax: 5, ZMin: -5, ZMax: 5},
	})
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	if _, _, err := mgr.JoinRoom(r.JoinCode, "P_001", "A", "Quest", nil); err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}

	first, err := mgr.SaveSceneMap(r.ID, SceneMapSavePayload{
		PlayerID:        "P_001",
		MapID:           "default",
		BaseVersion:     0,
		SchemaVersion:   "mrflag.scene.v1",
		CoordinateSpace: "shared_anchor",
		Map:             json.RawMessage(`{"objects":[{"id":"box"}]}`),
	})
	if err != nil {
		t.Fatalf("SaveSceneMap first: %v", err)
	}
	if first.Version != 1 {
		t.Fatalf("first version = %d, want 1", first.Version)
	}

	_, err = mgr.SaveSceneMap(r.ID, SceneMapSavePayload{
		PlayerID:    "P_001",
		MapID:       "default",
		BaseVersion: 0,
		Map:         json.RawMessage(`{"objects":[]}`),
	})
	if err == nil {
		t.Fatal("expected scene_map_conflict")
	}
	ce, ok := AsCodedError(err)
	if !ok || ce.Code != "scene_map_conflict" {
		t.Fatalf("err = %#v, want scene_map_conflict", err)
	}
}

func TestListRoomsIncludesPlayerCounts(t *testing.T) {
	mgr := NewManager(ManagerConfig{})
	first, err := mgr.CreateRoom(protocol.RoomConfig{RoomName: "alpha"})
	if err != nil {
		t.Fatalf("CreateRoom first: %v", err)
	}
	second, err := mgr.CreateRoom(protocol.RoomConfig{RoomName: "beta"})
	if err != nil {
		t.Fatalf("CreateRoom second: %v", err)
	}
	if _, _, err := mgr.JoinRoom(first.JoinCode, "P_001", "A", "Quest", nil); err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}

	rooms := mgr.ListRooms()
	if len(rooms) != 2 {
		t.Fatalf("len(ListRooms) = %d, want 2", len(rooms))
	}

	byID := map[string]RoomSummary{}
	for _, item := range rooms {
		byID[item.RoomID] = item
	}

	if got := byID[first.ID].PlayerCount; got != 1 {
		t.Fatalf("first player count = %d, want 1", got)
	}
	if got := byID[first.ID].MaxPlayers; got != 2 {
		t.Fatalf("first max players = %d, want 2", got)
	}
	if got := byID[second.ID].PlayerCount; got != 0 {
		t.Fatalf("second player count = %d, want 0", got)
	}
	if byID[first.ID].CreatedAt == 0 || byID[second.ID].CreatedAt == 0 {
		t.Fatal("created_at should be set")
	}
}
