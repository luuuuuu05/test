package room

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"math"
	mrand "math/rand/v2"
	"net"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"mrflag/pkg/protocol"
)

const (
	minPlayersToStart = 2
	maxPlayersPerRoom = 2
)

type Manager struct {
	mu            sync.RWMutex
	rooms         map[string]*Room
	byCode        map[string]string
	defaultRoomID string
	cfg           ManagerConfig
	events        EventSink
	seq           atomic.Uint32
}

func NewManager(cfg ManagerConfig) *Manager {
	if cfg.DefaultDuration <= 0 {
		cfg.DefaultDuration = 180
	}
	if cfg.DefaultMaxFlags <= 0 {
		cfg.DefaultMaxFlags = 5
	}
	if cfg.DefaultMinFlags <= 0 {
		cfg.DefaultMinFlags = 4
	}
	if cfg.RespawnDelay <= 0 {
		cfg.RespawnDelay = 2
	}
	if cfg.GrabDistance <= 0 {
		cfg.GrabDistance = 1.5
	}
	if cfg.DoubleDuration <= 0 {
		cfg.DoubleDuration = 20
	}
	if cfg.SceneMapMaxBytes <= 0 {
		cfg.SceneMapMaxBytes = 256 * 1024
	}
	if len(cfg.FlagWeights) == 0 {
		cfg.FlagWeights = map[string]int{"white": 50, "red": 35, "gold": 15}
	}
	return &Manager{
		rooms:  map[string]*Room{},
		byCode: map[string]string{},
		cfg:    cfg,
	}
}

func (m *Manager) SetEventSink(events EventSink) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = events
}

func (m *Manager) NextSeq() uint32 {
	return m.seq.Add(1)
}

func (m *Manager) CreateRoom(cfg protocol.RoomConfig) (*Room, error) {
	cfg = m.applyDefaults(cfg)
	room := &Room{
		ID:        "ROOM_" + token(6),
		Name:      cfg.RoomName,
		JoinCode:  token(6),
		Status:    StatusWaiting,
		Config:    cfg,
		Players:   map[string]*Player{},
		Flags:     map[string]*Flag{},
		CreatedAt: time.Now(),
	}
	if room.Name == "" {
		room.Name = room.ID
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for {
		if _, exists := m.rooms[room.ID]; !exists {
			break
		}
		room.ID = "ROOM_" + token(6)
	}
	for {
		if _, exists := m.byCode[room.JoinCode]; !exists {
			break
		}
		room.JoinCode = token(6)
	}
	m.rooms[room.ID] = room
	m.byCode[room.JoinCode] = room.ID
	return cloneRoom(room), nil
}

func (m *Manager) EnsureDefaultRoom() (*Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.defaultRoomLocked()
	log.Printf("default room ensured room=%s join_code=%s status=%s players=%d scene_exists=%v", r.ID, r.JoinCode, r.Status, len(r.Players), r.SceneMap != nil)
	return cloneRoom(r), nil
}

func (m *Manager) GetRoom(roomID string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil, false
	}
	return cloneRoom(r), true
}

func (m *Manager) ListRooms() []RoomSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]RoomSummary, 0, len(m.rooms))
	for _, r := range m.rooms {
		item := RoomSummary{
			RoomID:      r.ID,
			RoomName:    r.Name,
			JoinCode:    r.JoinCode,
			Status:      r.Status,
			PlayerCount: len(r.Players),
			MaxPlayers:  maxPlayersPerRoom,
			CreatedAt:   r.CreatedAt.UnixMilli(),
		}
		if !r.StartedAt.IsZero() {
			item.StartedAt = r.StartedAt.UnixMilli()
		}
		if !r.EndedAt.IsZero() {
			item.EndedAt = r.EndedAt.UnixMilli()
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt == out[j].CreatedAt {
			return out[i].RoomID < out[j].RoomID
		}
		return out[i].CreatedAt > out[j].CreatedAt
	})
	return out
}

func (m *Manager) Snapshot(roomID string) (*Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil, coded("room_not_found", "房间不存在", nil)
	}
	return snapshotLocked(r), nil
}

func (m *Manager) JoinRoom(joinCode, playerID, displayName, deviceType string, udpAddr *net.UDPAddr) (*Room, *Player, error) {
	if strings.TrimSpace(playerID) == "" {
		return nil, nil, coded("invalid_player_id", "player_id 不能为空", nil)
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = playerID
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	var r *Room
	joinCode = strings.ToUpper(strings.TrimSpace(joinCode))
	if joinCode == "" || joinCode == "DEFAULT" {
		r = m.defaultRoomLocked()
	} else {
		roomID, ok := m.byCode[joinCode]
		if !ok {
			return nil, nil, coded("invalid_join_code", "邀请码错误", nil)
		}
		r = m.rooms[roomID]
	}
	if r.Status == StatusEnded {
		return nil, nil, coded("game_ended", "本局已结束", nil)
	}

	if p, ok := r.Players[playerID]; ok {
		p.DisplayName = displayName
		p.DeviceType = deviceType
		if udpAddr != nil {
			p.UDPAddr = udpAddr
		}
		p.LastSeen = time.Now()
		return cloneRoom(r), clonePlayer(p), nil
	}
	if len(r.Players) >= maxPlayersPerRoom {
		return nil, nil, coded("room_full", "房间已满", nil)
	}

	slot := firstFreeSlot(r)
	p := &Player{
		ID:          playerID,
		DisplayName: displayName,
		DeviceType:  deviceType,
		Slot:        slot,
		SpawnPos:    spawnForSlot(r.Config.Bounds, slot),
		UDPAddr:     udpAddr,
		LastSeen:    time.Now(),
	}
	r.Players[playerID] = p
	r.PlayerOrder = append(r.PlayerOrder, playerID)
	return cloneRoom(r), clonePlayer(p), nil
}

func (m *Manager) AutoJoinDefaultPlayer(playerID, displayName, deviceType string, udpAddr *net.UDPAddr) (*Room, *Player, error) {
	autoGenerated := false
	if strings.TrimSpace(playerID) == "" {
		playerID = "AUTO_" + token(8)
		autoGenerated = true
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = playerID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.defaultRoomLocked()
	if p, ok := r.Players[playerID]; ok {
		p.DisplayName = displayName
		p.DeviceType = deviceType
		if udpAddr != nil {
			p.UDPAddr = udpAddr
		}
		p.LastSeen = time.Now()
		return cloneRoom(r), clonePlayer(p), nil
	}
	if len(r.Players) >= maxPlayersPerRoom && autoGenerated {
		log.Printf("default room full, removing oldest auto player room=%s current_players=%d incoming_player=%s", r.ID, len(r.Players), playerID)
		m.removeOldestAutoPlayerLocked(r)
	}
	if len(r.Players) >= maxPlayersPerRoom {
		return cloneRoom(r), nil, coded("room_full", "房间已满", nil)
	}
	slot := firstFreeSlot(r)
	p := &Player{
		ID:          playerID,
		DisplayName: displayName,
		DeviceType:  deviceType,
		Slot:        slot,
		SpawnPos:    spawnForSlot(r.Config.Bounds, slot),
		UDPAddr:     udpAddr,
		LastSeen:    time.Now(),
	}
	r.Players[playerID] = p
	r.PlayerOrder = append(r.PlayerOrder, playerID)
	log.Printf("default auto join stored room=%s player=%s slot=%d players=%d", r.ID, playerID, slot, len(r.Players))
	return cloneRoom(r), clonePlayer(p), nil
}

func (m *Manager) RebindPlayer(roomID, oldPlayerID, newPlayerID, displayName, deviceType string, udpAddr *net.UDPAddr) (*Room, *Player, error) {
	if strings.TrimSpace(newPlayerID) == "" {
		return nil, nil, coded("invalid_player_id", "player_id 不能为空", nil)
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = newPlayerID
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil, nil, coded("room_not_found", "房间不存在", nil)
	}
	if existing, ok := r.Players[newPlayerID]; ok {
		existing.DisplayName = displayName
		existing.DeviceType = deviceType
		if udpAddr != nil {
			existing.UDPAddr = udpAddr
		}
		existing.LastSeen = time.Now()
		if oldPlayerID != "" && oldPlayerID != newPlayerID {
			m.removePlayerLocked(r, oldPlayerID)
		}
		return cloneRoom(r), clonePlayer(existing), nil
	}
	old, ok := r.Players[oldPlayerID]
	if !ok {
		if len(r.Players) >= maxPlayersPerRoom {
			return nil, nil, coded("room_full", "房间已满", nil)
		}
		slot := firstFreeSlot(r)
		old = &Player{Slot: slot, SpawnPos: spawnForSlot(r.Config.Bounds, slot)}
	} else {
		delete(r.Players, oldPlayerID)
		for i, id := range r.PlayerOrder {
			if id == oldPlayerID {
				r.PlayerOrder[i] = newPlayerID
				break
			}
		}
	}
	old.ID = newPlayerID
	old.DisplayName = displayName
	old.DeviceType = deviceType
	if udpAddr != nil {
		old.UDPAddr = udpAddr
	}
	old.LastSeen = time.Now()
	r.Players[newPlayerID] = old
	if !containsString(r.PlayerOrder, newPlayerID) {
		r.PlayerOrder = append(r.PlayerOrder, newPlayerID)
	}
	return cloneRoom(r), clonePlayer(old), nil
}

func (m *Manager) RemovePlayer(roomID, playerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return
	}
	m.removePlayerLocked(r, playerID)
	if r.ID == m.defaultRoomID && r.Status == StatusEnded && len(r.Players) == 0 {
		r.Status = StatusWaiting
		r.Flags = map[string]*Flag{}
		r.StartedAt = time.Time{}
		r.EndedAt = time.Time{}
		r.generation++
	}
}

func (m *Manager) PlayerJoinedPayload(roomID, playerID string) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil, coded("room_not_found", "房间不存在", nil)
	}
	p, ok := r.Players[playerID]
	if !ok {
		return nil, coded("player_not_found", "玩家不存在", nil)
	}
	return map[string]any{
		"player_id":    p.ID,
		"display_name": p.DisplayName,
		"slot":         p.Slot,
		"spawn_pos":    p.SpawnPos,
		"player_count": len(r.Players),
		"ready":        len(r.Players) >= minPlayersToStart,
	}, nil
}

func (m *Manager) StartGame(roomID string) error {
	m.mu.Lock()
	r, ok := m.rooms[roomID]
	if !ok {
		m.mu.Unlock()
		return coded("room_not_found", "房间不存在", nil)
	}
	if r.Status == StatusPlaying || r.Status == StatusCountdown {
		m.mu.Unlock()
		return coded("game_already_started", "游戏已开始", nil)
	}
	if len(r.Players) < minPlayersToStart {
		m.mu.Unlock()
		return coded("not_enough_players", "需要 2 名玩家才能开始", map[string]any{"player_count": len(r.Players)})
	}
	r.Status = StatusCountdown
	r.Flags = map[string]*Flag{}
	r.StartedAt = time.Now()
	r.EndedAt = time.Time{}
	r.generation++
	generation := r.generation
	m.mu.Unlock()

	go m.runCountdown(roomID, generation)
	return nil
}

func (m *Manager) StopGame(roomID, reason string) error {
	if reason == "" {
		reason = "admin_abort"
	}
	return m.endGame(roomID, reason)
}

func (m *Manager) SetBounds(roomID string, bounds protocol.Bounds) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return coded("room_not_found", "房间不存在", nil)
	}
	if r.Status == StatusPlaying || r.Status == StatusCountdown {
		return coded("invalid_state", "游戏开始后不能修改场地边界", map[string]any{"status": r.Status})
	}
	r.Config.Bounds = bounds
	for _, p := range r.Players {
		p.SpawnPos = spawnForSlot(bounds, p.Slot)
	}
	return nil
}

func (m *Manager) GrabFlag(roomID, playerID, flagID string, pos protocol.Vector3) error {
	var (
		flag          *Flag
		scorePayload  any
		buffStart     any
		removePayload any
		respawnType   string
		generation    int64
		respawnDelay  float64
	)

	m.mu.Lock()
	r, ok := m.rooms[roomID]
	if !ok {
		m.mu.Unlock()
		return coded("room_not_found", "房间不存在", nil)
	}
	if r.Status != StatusPlaying {
		m.mu.Unlock()
		return coded("game_not_started", "游戏未开始", nil)
	}
	p, ok := r.Players[playerID]
	if !ok {
		m.mu.Unlock()
		return coded("player_not_found", "玩家不存在", nil)
	}
	flag, ok = r.Flags[flagID]
	if !ok {
		m.mu.Unlock()
		return coded("flag_not_exist", "旗帜不存在或已被取走", nil)
	}
	distance := dist(pos, flag.Pos)
	if distance > m.cfg.GrabDistance {
		m.mu.Unlock()
		return coded("grab_too_far", "距离超出抓取范围", map[string]any{
			"distance":     math.Round(distance*100) / 100,
			"max_distance": m.cfg.GrabDistance,
		})
	}

	delete(r.Flags, flagID)
	p.LastPos = pos
	p.LastSeen = time.Now()
	removePayload = map[string]any{
		"flag_id":    flag.ID,
		"reason":     "grabbed",
		"grabbed_by": playerID,
	}

	if flag.Type == "double" {
		p.DoubleUntil = time.Now().Add(time.Duration(m.cfg.DoubleDuration) * time.Second)
		buffStart = map[string]any{
			"player_id": playerID,
			"buff_type": "double_score",
			"duration":  m.cfg.DoubleDuration,
			"end_ts":    p.DoubleUntil.UnixMilli(),
		}
	} else {
		delta := flag.Score
		doubleActive := time.Now().Before(p.DoubleUntil)
		if doubleActive {
			delta *= 2
		}
		p.Score += delta
		p.FlagsGrabbed++
		scorePayload = map[string]any{
			"player_id":     playerID,
			"delta":         delta,
			"total":         p.Score,
			"double_active": doubleActive,
			"scoreboard":    scoreboardLocked(r),
		}
	}
	respawnType = flag.Type
	generation = r.generation
	respawnDelay = r.Config.RespawnDelay
	if respawnDelay <= 0 {
		respawnDelay = m.cfg.RespawnDelay
	}
	m.mu.Unlock()

	m.broadcast(roomID, protocol.TypeFlagRemove, removePayload)
	if scorePayload != nil {
		m.broadcast(roomID, protocol.TypeScoreUpdate, scorePayload)
	}
	if buffStart != nil {
		m.broadcast(roomID, protocol.TypeBuffStart, buffStart)
		go m.expireBuff(roomID, playerID, generation, time.Duration(m.cfg.DoubleDuration)*time.Second)
	}
	go m.respawnLater(roomID, generation, respawnType, respawnDelay)
	return nil
}

func (m *Manager) HandleUDPPacket(pkt protocol.PosPacket, addr *net.UDPAddr) []protocol.RelayPacket {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[pkt.RoomID]
	if !ok {
		var found *Room
		for _, candidate := range m.rooms {
			if _, exists := candidate.Players[pkt.PlayerID]; exists {
				found = candidate
				break
			}
		}
		if found == nil {
			log.Printf("udp drop unknown room/player pkt_room=%s player=%s from=%s", pkt.RoomID, pkt.PlayerID, addr)
			return nil
		}
		log.Printf("udp room overridden pkt_room=%s bound_room=%s player=%s from=%s", pkt.RoomID, found.ID, pkt.PlayerID, addr)
		r = found
	}
	p, ok := r.Players[pkt.PlayerID]
	if !ok {
		log.Printf("udp drop player not in room room=%s player=%s from=%s", r.ID, pkt.PlayerID, addr)
		return nil
	}
	p.UDPAddr = addr
	p.LastPos = protocol.Vector3{X: float64(pkt.X), Y: float64(pkt.Y), Z: float64(pkt.Z)}
	p.LastSeen = time.Now()

	relay := protocol.RelayPacket{
		FromPlayerID: pkt.PlayerID,
		Seq:          pkt.Seq,
		ServerTs:     time.Now().UnixMilli(),
		X:            pkt.X,
		Y:            pkt.Y,
		Z:            pkt.Z,
		RotY:         pkt.RotY,
		HeadPitch:    pkt.HeadPitch,
		Flags:        pkt.Flags,
	}
	packets := make([]protocol.RelayPacket, 0, len(r.Players)-1)
	for id, other := range r.Players {
		if id == pkt.PlayerID || other.UDPAddr == nil {
			continue
		}
		packets = append(packets, relay)
	}
	if len(packets) == 0 {
		log.Printf("udp no recipients room=%s player=%s players=%d from=%s", r.ID, pkt.PlayerID, len(r.Players), addr)
	}
	return packets
}

func (m *Manager) UDPRecipients(roomID, fromPlayerID string) []*net.UDPAddr {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	if !ok {
		r = nil
		for _, candidate := range m.rooms {
			if _, exists := candidate.Players[fromPlayerID]; exists {
				r = candidate
				break
			}
		}
		if r == nil {
			return nil
		}
	} else if _, exists := r.Players[fromPlayerID]; !exists {
		for _, candidate := range m.rooms {
			if _, found := candidate.Players[fromPlayerID]; found {
				r = candidate
				break
			}
		}
	}
	addrs := make([]*net.UDPAddr, 0, len(r.Players)-1)
	for id, p := range r.Players {
		if id == fromPlayerID || p.UDPAddr == nil {
			continue
		}
		addrs = append(addrs, p.UDPAddr)
	}
	return addrs
}

func (m *Manager) Scores(roomID string) ([]PlayerScore, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil, coded("room_not_found", "房间不存在", nil)
	}
	return scoreboardLocked(r), nil
}

func (m *Manager) Flags(roomID string) ([]*Flag, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil, coded("room_not_found", "房间不存在", nil)
	}
	return flagsLocked(r), nil
}

func (m *Manager) SaveSceneMap(roomID string, payload SceneMapSavePayload) (*SceneMap, error) {
	if strings.TrimSpace(payload.PlayerID) == "" {
		return nil, coded("invalid_scene_map", "player_id 不能为空", nil)
	}
	if len(payload.Map) == 0 || !json.Valid(payload.Map) {
		return nil, coded("invalid_scene_map", "map 必须是合法 JSON 对象/数组", nil)
	}
	if len(payload.Map) > m.cfg.SceneMapMaxBytes {
		return nil, coded("scene_map_too_large", "场景地图 JSON 过大", map[string]any{
			"size":      len(payload.Map),
			"max_bytes": m.cfg.SceneMapMaxBytes,
		})
	}
	if payload.MapID == "" {
		payload.MapID = "default"
	}
	if payload.CoordinateSpace == "" {
		payload.CoordinateSpace = "shared_anchor"
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil, coded("room_not_found", "房间不存在", nil)
	}
	if _, ok := r.Players[payload.PlayerID]; !ok && payload.PlayerID != "admin" {
		return nil, coded("player_not_found", "玩家不存在", nil)
	}
	currentVersion := uint64(0)
	if r.SceneMap != nil {
		currentVersion = r.SceneMap.Version
	}
	if !payload.Force && payload.BaseVersion != currentVersion {
		return nil, coded("scene_map_conflict", "场景地图版本冲突", map[string]any{
			"current_version": currentVersion,
			"base_version":    payload.BaseVersion,
		})
	}
	saved := &SceneMap{
		MapID:           payload.MapID,
		Version:         currentVersion + 1,
		SchemaVersion:   payload.SchemaVersion,
		AnchorID:        payload.AnchorID,
		CoordinateSpace: payload.CoordinateSpace,
		UpdatedBy:       payload.PlayerID,
		UpdatedTs:       time.Now().UnixMilli(),
		Map:             append(json.RawMessage(nil), payload.Map...),
	}
	r.SceneMap = saved
	log.Printf("scene map stored room=%s map_id=%s version=%d updated_by=%s bytes=%d", roomID, saved.MapID, saved.Version, saved.UpdatedBy, len(saved.Map))
	return cloneSceneMap(saved), nil
}

func (m *Manager) SceneMap(roomID string) (*SceneMap, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return nil, coded("room_not_found", "房间不存在", nil)
	}
	return cloneSceneMap(r.SceneMap), nil
}

func (m *Manager) runCountdown(roomID string, generation int64) {
	for _, count := range []int{3, 2, 1, 0} {
		if !m.isGeneration(roomID, generation, StatusCountdown) {
			return
		}
		m.broadcastPlayers(roomID, protocol.TypeGameCountdown, map[string]any{"count": count})
		if count > 0 {
			time.Sleep(time.Second)
		}
	}

	var spawned []*Flag
	duration := 0
	m.mu.Lock()
	r, ok := m.rooms[roomID]
	if !ok || r.generation != generation || r.Status != StatusCountdown {
		m.mu.Unlock()
		return
	}
	r.Status = StatusPlaying
	r.StartedAt = time.Now()
	duration = r.Config.GameDuration
	spawned = m.fillFlagsLocked(r, true)
	spawned = append(spawned, m.spawnDoubleItemsLocked(r)...)
	m.mu.Unlock()

	m.broadcast(roomID, protocol.TypeGameStart, map[string]any{
		"duration": duration,
		"end_ts":   time.Now().Add(time.Duration(duration) * time.Second).UnixMilli(),
	})
	for _, f := range spawned {
		m.broadcast(roomID, protocol.TypeFlagSpawn, f)
	}

	time.Sleep(time.Duration(duration) * time.Second)
	_ = m.endGameIfGeneration(roomID, "timeout", generation)
}

func (m *Manager) endGame(roomID, reason string) error {
	m.mu.RLock()
	r, ok := m.rooms[roomID]
	if !ok {
		m.mu.RUnlock()
		return coded("room_not_found", "房间不存在", nil)
	}
	generation := r.generation
	m.mu.RUnlock()
	return m.endGameIfGeneration(roomID, reason, generation)
}

func (m *Manager) endGameIfGeneration(roomID, reason string, generation int64) error {
	var payload any
	var removed []*Flag

	m.mu.Lock()
	r, ok := m.rooms[roomID]
	if !ok {
		m.mu.Unlock()
		return coded("room_not_found", "房间不存在", nil)
	}
	if r.generation != generation {
		m.mu.Unlock()
		return nil
	}
	if r.Status == StatusEnded || r.Status == StatusWaiting {
		m.mu.Unlock()
		return nil
	}
	now := time.Now()
	durationActual := 0
	if !r.StartedAt.IsZero() {
		durationActual = int(now.Sub(r.StartedAt).Seconds())
	}
	r.Status = StatusEnded
	r.EndedAt = now
	r.generation++
	for _, p := range r.Players {
		if !p.DoubleUntil.IsZero() && now.Before(p.DoubleUntil) {
			p.DoubleUntil = time.Time{}
		}
	}
	for _, f := range r.Flags {
		removed = append(removed, cloneFlag(f))
	}
	r.Flags = map[string]*Flag{}
	payload = map[string]any{
		"reason":          reason,
		"duration_actual": durationActual,
		"final_scores":    finalScoresLocked(r),
	}
	m.mu.Unlock()

	for _, f := range removed {
		m.broadcast(roomID, protocol.TypeFlagRemove, map[string]any{
			"flag_id": f.ID,
			"reason":  "game_end",
		})
	}
	m.broadcast(roomID, protocol.TypeGameEnd, payload)
	return nil
}

func (m *Manager) expireBuff(roomID, playerID string, generation int64, after time.Duration) {
	time.Sleep(after)
	m.mu.Lock()
	r, ok := m.rooms[roomID]
	if !ok || r.generation != generation || r.Status != StatusPlaying {
		m.mu.Unlock()
		return
	}
	p, ok := r.Players[playerID]
	if !ok || time.Now().Before(p.DoubleUntil) {
		m.mu.Unlock()
		return
	}
	p.DoubleUntil = time.Time{}
	m.mu.Unlock()

	m.broadcast(roomID, protocol.TypeBuffEnd, map[string]any{
		"player_id": playerID,
		"buff_type": "double_score",
		"reason":    "expired",
	})
}

func (m *Manager) respawnLater(roomID string, generation int64, removedType string, delaySeconds float64) {
	time.Sleep(time.Duration(delaySeconds * float64(time.Second)))
	var spawned []*Flag
	m.mu.Lock()
	r, ok := m.rooms[roomID]
	if !ok || r.generation != generation || r.Status != StatusPlaying {
		m.mu.Unlock()
		return
	}
	if removedType == "double" {
		spawned = append(spawned, m.spawnDoubleItemsLocked(r)...)
	} else {
		spawned = append(spawned, m.fillFlagsLocked(r, false)...)
	}
	m.mu.Unlock()
	for _, f := range spawned {
		m.broadcast(roomID, protocol.TypeFlagSpawn, f)
	}
}

func (m *Manager) isGeneration(roomID string, generation int64, status Status) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	return ok && r.generation == generation && r.Status == status
}

func (m *Manager) broadcast(roomID, typ string, payload any) {
	m.mu.RLock()
	events := m.events
	m.mu.RUnlock()
	if events == nil {
		return
	}
	events.Broadcast(roomID, protocol.NewMessage(typ, m.NextSeq(), roomID, payload))
}

func (m *Manager) broadcastPlayers(roomID, typ string, payload any) {
	m.mu.RLock()
	events := m.events
	m.mu.RUnlock()
	if events == nil {
		return
	}
	events.BroadcastPlayers(roomID, protocol.NewMessage(typ, m.NextSeq(), roomID, payload))
}

func (m *Manager) fillFlagsLocked(r *Room, reset bool) []*Flag {
	if reset {
		r.Flags = map[string]*Flag{}
	}
	normalCount := 0
	for _, f := range r.Flags {
		if f.Type != "double" {
			normalCount++
		}
	}
	target := r.Config.MaxFlags
	if target <= 0 {
		target = m.cfg.DefaultMaxFlags
	}
	spawned := []*Flag{}
	for normalCount < target {
		f := m.spawnFlagLocked(r, weightedFlagType(m.cfg.FlagWeights))
		if f == nil {
			break
		}
		spawned = append(spawned, f)
		normalCount++
	}
	return spawned
}

func (m *Manager) spawnDoubleItemsLocked(r *Room) []*Flag {
	maxDouble := r.Config.MaxDoubleItems
	if maxDouble < 0 {
		maxDouble = 0
	}
	if maxDouble == 0 {
		maxDouble = m.cfg.MaxDoubleItems
	}
	active := 0
	for _, f := range r.Flags {
		if f.Type == "double" {
			active++
		}
	}
	spawned := []*Flag{}
	for active < maxDouble {
		f := m.spawnFlagLocked(r, "double")
		if f == nil {
			break
		}
		spawned = append(spawned, f)
		active++
	}
	return spawned
}

func (m *Manager) spawnFlagLocked(r *Room, typ string) *Flag {
	pointID, pos, ok := m.nextFreePointLocked(r)
	if !ok {
		return nil
	}
	score := 0
	switch typ {
	case "white":
		score = 10
	case "red":
		score = 20
	case "gold":
		score = 30
	case "double":
		score = 0
	default:
		typ = "white"
		score = 10
	}
	f := &Flag{
		ID:          "FLAG_" + token(4),
		Type:        typ,
		Score:       score,
		Pos:         pos,
		FlagPointID: pointID,
		Lifetime:    -1,
		CreatedAt:   time.Now().UnixMilli(),
	}
	for {
		if _, exists := r.Flags[f.ID]; !exists {
			break
		}
		f.ID = "FLAG_" + token(4)
	}
	r.Flags[f.ID] = f
	return cloneFlag(f)
}

func (m *Manager) nextFreePointLocked(r *Room) (string, protocol.Vector3, bool) {
	used := map[string]bool{}
	for _, f := range r.Flags {
		if f.FlagPointID != "" {
			used[f.FlagPointID] = true
		}
	}
	points := r.Config.FlagPoints
	if len(points) == 0 {
		r.autoPoint++
		x := r.Config.Bounds.XMin + mrand.Float64()*(r.Config.Bounds.XMax-r.Config.Bounds.XMin)
		z := r.Config.Bounds.ZMin + mrand.Float64()*(r.Config.Bounds.ZMax-r.Config.Bounds.ZMin)
		return "AUTO_" + strconvID(r.autoPoint), protocol.Vector3{X: x, Y: 0, Z: z}, true
	}
	for i := 0; i < len(points); i++ {
		idx := (r.autoPoint + i) % len(points)
		fp := points[idx]
		if used[fp.ID] {
			continue
		}
		r.autoPoint = idx + 1
		return fp.ID, protocol.Vector3{X: fp.X, Y: fp.Y, Z: fp.Z}, true
	}
	return "", protocol.Vector3{}, false
}

func (m *Manager) applyDefaults(cfg protocol.RoomConfig) protocol.RoomConfig {
	if cfg.GameDuration <= 0 {
		cfg.GameDuration = m.cfg.DefaultDuration
	}
	if cfg.MaxFlags <= 0 {
		cfg.MaxFlags = m.cfg.DefaultMaxFlags
	}
	if cfg.MinFlags <= 0 {
		cfg.MinFlags = m.cfg.DefaultMinFlags
	}
	if cfg.RespawnDelay <= 0 {
		cfg.RespawnDelay = m.cfg.RespawnDelay
	}
	if cfg.MaxDoubleItems < 0 {
		cfg.MaxDoubleItems = 0
	} else if cfg.MaxDoubleItems == 0 {
		cfg.MaxDoubleItems = m.cfg.MaxDoubleItems
	}
	if cfg.Bounds.XMin == 0 && cfg.Bounds.XMax == 0 && cfg.Bounds.ZMin == 0 && cfg.Bounds.ZMax == 0 {
		cfg.Bounds = protocol.Bounds{XMin: -5, XMax: 5, ZMin: -5, ZMax: 5}
	}
	return cfg
}

func (m *Manager) defaultRoomLocked() *Room {
	if m.defaultRoomID != "" {
		if r := m.rooms[m.defaultRoomID]; r != nil {
			if r.Status == StatusEnded && len(r.Players) == 0 {
				r.Status = StatusWaiting
				r.Flags = map[string]*Flag{}
				r.StartedAt = time.Time{}
				r.EndedAt = time.Time{}
				r.generation++
			}
			return r
		}
	}
	cfg := m.applyDefaults(protocol.RoomConfig{
		RoomName: "DefaultRoom",
		Bounds:   protocol.Bounds{XMin: -5, XMax: 5, ZMin: -5, ZMax: 5},
		FlagPoints: []protocol.FlagPoint{
			{ID: "FP_01", X: 1.2, Y: 0, Z: 2.3},
			{ID: "FP_02", X: -2.1, Y: 0, Z: 1.5},
			{ID: "FP_03", X: 0.5, Y: 0, Z: -3.0},
			{ID: "FP_04", X: 3.0, Y: 0, Z: 0.8},
			{ID: "FP_05", X: -1.8, Y: 0, Z: -2.2},
		},
	})
	r := &Room{
		ID:        "ROOM_DEFAULT",
		Name:      "DefaultRoom",
		JoinCode:  "DEFAULT",
		Status:    StatusWaiting,
		Config:    cfg,
		Players:   map[string]*Player{},
		Flags:     map[string]*Flag{},
		CreatedAt: time.Now(),
	}
	m.rooms[r.ID] = r
	m.byCode[r.JoinCode] = r.ID
	m.defaultRoomID = r.ID
	log.Printf("default room created room=%s join_code=%s max_flags=%d flag_points=%d", r.ID, r.JoinCode, r.Config.MaxFlags, len(r.Config.FlagPoints))
	return r
}

func (m *Manager) removePlayerLocked(r *Room, playerID string) {
	if r == nil || playerID == "" {
		return
	}
	delete(r.Players, playerID)
	for i, id := range r.PlayerOrder {
		if id == playerID {
			r.PlayerOrder = append(r.PlayerOrder[:i], r.PlayerOrder[i+1:]...)
			break
		}
	}
}

func (m *Manager) removeOldestAutoPlayerLocked(r *Room) {
	for _, playerID := range append([]string(nil), r.PlayerOrder...) {
		if strings.HasPrefix(playerID, "AUTO_") {
			log.Printf("removing auto player room=%s player=%s", r.ID, playerID)
			m.removePlayerLocked(r, playerID)
			return
		}
	}
}

func (m *Manager) RoomInfo(roomID string) (protocol.RoomInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[roomID]
	if !ok {
		return protocol.RoomInfo{}, coded("room_not_found", "房间不存在", nil)
	}
	return protocol.RoomInfo{
		RoomID:   r.ID,
		JoinCode: r.JoinCode,
		Status:   string(r.Status),
	}, nil
}

func token(n int) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return strings.ToUpper(hex.EncodeToString(b[:])[:n])
	}
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, n)
	for i := range out {
		out[i] = alphabet[mrand.IntN(len(alphabet))]
	}
	return string(out)
}

func strconvID(n int) string {
	if n == 0 {
		return "0"
	}
	const digits = "0123456789"
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	return string(buf[i:])
}

func weightedFlagType(weights map[string]int) string {
	total := 0
	for _, k := range []string{"white", "red", "gold"} {
		total += max(weights[k], 0)
	}
	if total <= 0 {
		return "white"
	}
	pick := mrand.IntN(total)
	for _, k := range []string{"white", "red", "gold"} {
		w := max(weights[k], 0)
		if pick < w {
			return k
		}
		pick -= w
	}
	return "white"
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func firstFreeSlot(r *Room) int {
	used := map[int]bool{}
	for _, p := range r.Players {
		used[p.Slot] = true
	}
	if !used[0] {
		return 0
	}
	return 1
}

func spawnForSlot(bounds protocol.Bounds, slot int) protocol.Vector3 {
	z := (bounds.ZMin + bounds.ZMax) / 2
	y := 0.0
	if slot == 0 {
		return protocol.Vector3{X: bounds.XMin + 2, Y: y, Z: z}
	}
	return protocol.Vector3{X: bounds.XMax - 2, Y: y, Z: z}
}

func dist(a, b protocol.Vector3) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func scoreboardLocked(r *Room) []PlayerScore {
	out := make([]PlayerScore, 0, len(r.Players))
	for _, p := range r.Players {
		out = append(out, PlayerScore{
			PlayerID:    p.ID,
			DisplayName: p.DisplayName,
			Score:       p.Score,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].PlayerID < out[j].PlayerID
		}
		return out[i].Score > out[j].Score
	})
	return out
}

func finalScoresLocked(r *Room) []FinalScore {
	players := make([]*Player, 0, len(r.Players))
	for _, p := range r.Players {
		players = append(players, p)
	}
	sort.Slice(players, func(i, j int) bool {
		if players[i].Score == players[j].Score {
			return players[i].ID < players[j].ID
		}
		return players[i].Score > players[j].Score
	})
	out := make([]FinalScore, 0, len(players))
	for i, p := range players {
		award := "runner_up"
		if i == 0 {
			award = "champion"
		}
		out = append(out, FinalScore{
			Rank:         i + 1,
			PlayerID:     p.ID,
			DisplayName:  p.DisplayName,
			Score:        p.Score,
			FlagsGrabbed: p.FlagsGrabbed,
			Award:        award,
		})
	}
	return out
}

func flagsLocked(r *Room) []*Flag {
	out := make([]*Flag, 0, len(r.Flags))
	for _, f := range r.Flags {
		out = append(out, cloneFlag(f))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func snapshotLocked(r *Room) *Snapshot {
	players := make([]*Player, 0, len(r.Players))
	for _, id := range r.PlayerOrder {
		if p, ok := r.Players[id]; ok {
			players = append(players, clonePlayer(p))
		}
	}
	s := &Snapshot{
		RoomID:     r.ID,
		RoomName:   r.Name,
		JoinCode:   r.JoinCode,
		Status:     r.Status,
		Config:     r.Config,
		Players:    players,
		Flags:      flagsLocked(r),
		Scoreboard: scoreboardLocked(r),
		SceneMap:   cloneSceneMap(r.SceneMap),
		CreatedAt:  r.CreatedAt.UnixMilli(),
	}
	if !r.StartedAt.IsZero() {
		s.StartedAt = r.StartedAt.UnixMilli()
	}
	if !r.EndedAt.IsZero() {
		s.EndedAt = r.EndedAt.UnixMilli()
	}
	return s
}

func cloneRoom(r *Room) *Room {
	if r == nil {
		return nil
	}
	cp := *r
	cp.Players = map[string]*Player{}
	for k, v := range r.Players {
		cp.Players[k] = clonePlayer(v)
	}
	cp.PlayerOrder = append([]string(nil), r.PlayerOrder...)
	cp.Flags = map[string]*Flag{}
	for k, v := range r.Flags {
		cp.Flags[k] = cloneFlag(v)
	}
	cp.SceneMap = cloneSceneMap(r.SceneMap)
	cp.Config.FlagPoints = append([]protocol.FlagPoint(nil), r.Config.FlagPoints...)
	return &cp
}

func clonePlayer(p *Player) *Player {
	if p == nil {
		return nil
	}
	cp := *p
	if p.UDPAddr != nil {
		addr := *p.UDPAddr
		cp.UDPAddr = &addr
	}
	return &cp
}

func cloneFlag(f *Flag) *Flag {
	if f == nil {
		return nil
	}
	cp := *f
	return &cp
}

func cloneSceneMap(s *SceneMap) *SceneMap {
	if s == nil {
		return nil
	}
	cp := *s
	cp.Map = append(json.RawMessage(nil), s.Map...)
	return &cp
}

func AsCodedError(err error) (*CodedError, bool) {
	var ce *CodedError
	if errors.As(err, &ce) {
		return ce, true
	}
	return nil, false
}
