package room

import (
	"encoding/json"
	"net"
	"time"

	"mrflag/pkg/protocol"
)

type Status string

const (
	StatusWaiting   Status = "waiting"
	StatusCountdown Status = "countdown"
	StatusPlaying   Status = "playing"
	StatusEnded     Status = "ended"
)

type EventSink interface {
	Broadcast(roomID string, msg protocol.Message)
	BroadcastPlayers(roomID string, msg protocol.Message)
	BroadcastPlayersExcept(roomID, playerID string, msg protocol.Message)
	SendPlayer(roomID, playerID string, msg protocol.Message)
}

type ManagerConfig struct {
	DefaultDuration  int
	DefaultMaxFlags  int
	DefaultMinFlags  int
	RespawnDelay     float64
	GrabDistance     float64
	DoubleDuration   int
	MaxDoubleItems   int
	SceneMapMaxBytes int
	FlagWeights      map[string]int
}

type Room struct {
	ID       string
	Name     string
	JoinCode string
	Status   Status
	Config   protocol.RoomConfig

	Players     map[string]*Player
	PlayerOrder []string
	Flags       map[string]*Flag
	SceneMap    *SceneMap

	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time

	generation int64
	autoPoint  int
}

type RoomSummary struct {
	RoomID      string `json:"room_id"`
	RoomName    string `json:"room_name"`
	JoinCode    string `json:"join_code"`
	Status      Status `json:"status"`
	PlayerCount int    `json:"player_count"`
	MaxPlayers  int    `json:"max_players"`
	CreatedAt   int64  `json:"created_at"`
	StartedAt   int64  `json:"started_at,omitempty"`
	EndedAt     int64  `json:"ended_at,omitempty"`
}

type Player struct {
	ID           string           `json:"player_id"`
	DisplayName  string           `json:"display_name"`
	DeviceType   string           `json:"device_type"`
	Slot         int              `json:"slot"`
	SpawnPos     protocol.Vector3 `json:"spawn_pos"`
	Score        int              `json:"score"`
	FlagsGrabbed int              `json:"flags_grabbed"`
	DoubleUntil  time.Time        `json:"-"`
	UDPAddr      *net.UDPAddr     `json:"-"`
	LastPos      protocol.Vector3 `json:"last_pos"`
	LastSeen     time.Time        `json:"last_seen"`
}

type Flag struct {
	ID          string           `json:"flag_id"`
	Type        string           `json:"flag_type"`
	Score       int              `json:"score"`
	Pos         protocol.Vector3 `json:"pos"`
	FlagPointID string           `json:"flag_point_id"`
	Lifetime    int              `json:"lifetime"`
	CreatedAt   int64            `json:"created_at"`
}

type PlayerScore struct {
	PlayerID    string `json:"player_id"`
	DisplayName string `json:"display_name"`
	Score       int    `json:"score"`
}

type FinalScore struct {
	Rank         int    `json:"rank"`
	PlayerID     string `json:"player_id"`
	DisplayName  string `json:"display_name"`
	Score        int    `json:"score"`
	FlagsGrabbed int    `json:"flags_grabbed"`
	Award        string `json:"award"`
}

type SceneMap struct {
	MapID           string          `json:"map_id"`
	Version         uint64          `json:"version"`
	SchemaVersion   string          `json:"schema_version,omitempty"`
	AnchorID        string          `json:"anchor_id,omitempty"`
	CoordinateSpace string          `json:"coordinate_space,omitempty"`
	UpdatedBy       string          `json:"updated_by"`
	UpdatedTs       int64           `json:"updated_ts"`
	Map             json.RawMessage `json:"map"`
}

type SceneMapSavePayload struct {
	PlayerID        string          `json:"player_id"`
	MapID           string          `json:"map_id"`
	BaseVersion     uint64          `json:"base_version"`
	Force           bool            `json:"force"`
	SchemaVersion   string          `json:"schema_version"`
	AnchorID        string          `json:"anchor_id"`
	CoordinateSpace string          `json:"coordinate_space"`
	Map             json.RawMessage `json:"map"`
}

type Snapshot struct {
	RoomID     string              `json:"room_id"`
	RoomName   string              `json:"room_name"`
	JoinCode   string              `json:"join_code"`
	Status     Status              `json:"status"`
	Config     protocol.RoomConfig `json:"config"`
	Players    []*Player           `json:"players"`
	Flags      []*Flag             `json:"flags"`
	Scoreboard []PlayerScore       `json:"scoreboard"`
	SceneMap   *SceneMap           `json:"scene_map,omitempty"`
	CreatedAt  int64               `json:"created_at"`
	StartedAt  int64               `json:"started_at,omitempty"`
	EndedAt    int64               `json:"ended_at,omitempty"`
}
