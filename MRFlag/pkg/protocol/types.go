package protocol

import (
	"encoding/json"
	"time"
)

const (
	TypeCreateRoom       = "c_create_room"
	TypeStartGame        = "c_start_game"
	TypeStopGame         = "c_stop_game"
	TypeSetBounds        = "c_set_bounds"
	TypePlayerJoin       = "c_player_join"
	TypeGrabFlag         = "c_grab_flag"
	TypeHeartbeat        = "c_heartbeat"
	TypeSceneMapSave     = "c_scene_map_save"
	TypeSceneMapRequest  = "c_scene_map_request"
	TypeRoomCreated      = "s_room_created"
	TypePlayerJoined     = "s_player_joined"
	TypeGameCountdown    = "s_game_countdown"
	TypeGameStart        = "s_game_start"
	TypeGameEnd          = "s_game_end"
	TypeFlagSpawn        = "s_flag_spawn"
	TypeFlagRemove       = "s_flag_remove"
	TypeScoreUpdate      = "s_score_update"
	TypeBuffStart        = "s_buff_start"
	TypeBuffEnd          = "s_buff_end"
	TypePlayerState      = "s_player_state"
	TypeError            = "s_error"
	TypeSceneMapSaved    = "s_scene_map_saved"
	TypeSceneMapUpdate   = "s_scene_map_update"
	TypeSceneMapSnapshot = "s_scene_map_snapshot"
)

type IncomingMessage struct {
	Type    string          `json:"type"`
	Seq     uint32          `json:"seq"`
	Ts      int64           `json:"ts"`
	RoomID  string          `json:"room_id"`
	Payload json.RawMessage `json:"payload"`
}

type Message struct {
	Type    string `json:"type"`
	Seq     uint32 `json:"seq"`
	Ts      int64  `json:"ts"`
	RoomID  string `json:"room_id"`
	Payload any    `json:"payload"`
}

func NewMessage(msgType string, seq uint32, roomID string, payload any) Message {
	return Message{
		Type:    msgType,
		Seq:     seq,
		Ts:      time.Now().UnixMilli(),
		RoomID:  roomID,
		Payload: payload,
	}
}

type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Bounds struct {
	XMin float64 `json:"x_min"`
	XMax float64 `json:"x_max"`
	ZMin float64 `json:"z_min"`
	ZMax float64 `json:"z_max"`
}

type FlagPoint struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	Z  float64 `json:"z"`
}

type RoomConfig struct {
	RoomName       string      `json:"room_name"`
	GameDuration   int         `json:"game_duration"`
	MaxFlags       int         `json:"max_flags"`
	MinFlags       int         `json:"min_flags"`
	RespawnDelay   float64     `json:"respawn_delay"`
	MaxDoubleItems int         `json:"max_double_items"`
	Bounds         Bounds      `json:"bounds"`
	FlagPoints     []FlagPoint `json:"flag_points"`
}

type RoomInfo struct {
	RoomID   string `json:"room_id"`
	JoinCode string `json:"join_code"`
	Status   string `json:"status"`
}

type PlayerJoinPayload struct {
	PlayerID    string `json:"player_id"`
	JoinCode    string `json:"join_code"`
	DisplayName string `json:"display_name"`
	DeviceType  string `json:"device_type"`
	UDPPort     int    `json:"udp_port"`
}

type GrabFlagPayload struct {
	PlayerID string  `json:"player_id"`
	FlagID   string  `json:"flag_id"`
	Pos      Vector3 `json:"pos"`
}

type StopGamePayload struct {
	Reason string `json:"reason"`
}

type SetBoundsPayload struct {
	Bounds Bounds `json:"bounds"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
}

type PosPacket struct {
	RoomID    string
	PlayerID  string
	Seq       uint32
	Ts        int64
	X         float32
	Y         float32
	Z         float32
	RotY      float32
	HeadPitch float32
	Flags     uint8
}

type RelayPacket struct {
	FromPlayerID string
	Seq          uint32
	ServerTs     int64
	X            float32
	Y            float32
	Z            float32
	RotY         float32
	HeadPitch    float32
	Flags        uint8
}
