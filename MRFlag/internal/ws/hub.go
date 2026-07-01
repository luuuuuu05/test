package ws

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"mrflag/internal/room"
	"mrflag/internal/wsconn"
	"mrflag/pkg/protocol"
)

type Hub struct {
	mu      sync.RWMutex
	mgr     *room.Manager
	clients map[*Client]struct{}
	rooms   map[string]map[*Client]struct{}
	players map[string]map[string]*Client
}

type Client struct {
	hub        *Hub
	conn       *wsconn.Conn
	clientType string
	roomID     string
	playerID   string
	send       chan protocol.Message
	done       chan struct{}
	closeOnce  sync.Once
}

func NewHub(mgr *room.Manager) *Hub {
	return &Hub{
		mgr:     mgr,
		clients: map[*Client]struct{}{},
		rooms:   map[string]map[*Client]struct{}{},
		players: map[string]map[string]*Client{},
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsconn.Upgrade(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	clientType := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Client-Type")))
	if clientType == "" {
		clientType = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("client_type")))
	}
	if clientType != "admin" {
		clientType = "player"
	}
	c := &Client{
		hub:        h,
		conn:       conn,
		clientType: clientType,
		send:       make(chan protocol.Message, 64),
		done:       make(chan struct{}),
	}
	roomID := strings.TrimSpace(r.Header.Get("X-Room-ID"))
	if roomID == "" {
		roomID = strings.TrimSpace(r.URL.Query().Get("room_id"))
	}

	h.register(c)
	if roomID != "" {
		h.bindRoom(c, roomID)
	}
	go c.writeLoop()
	if c.clientType == "player" {
		c.autoJoinDefault(r)
	}
	c.readLoop()
}

func (h *Hub) Broadcast(roomID string, msg protocol.Message) {
	clients := h.roomClients(roomID, false, "")
	if shouldLogWSMessage(msg.Type) {
		log.Printf("ws broadcast type=%s room=%s recipients=%d", msg.Type, roomID, len(clients))
	}
	for _, c := range clients {
		c.Send(msg)
	}
}

func (h *Hub) BroadcastPlayers(roomID string, msg protocol.Message) {
	clients := h.roomClients(roomID, true, "")
	if shouldLogWSMessage(msg.Type) {
		log.Printf("ws broadcast players type=%s room=%s recipients=%d", msg.Type, roomID, len(clients))
	}
	for _, c := range clients {
		c.Send(msg)
	}
}

func (h *Hub) BroadcastPlayersExcept(roomID, playerID string, msg protocol.Message) {
	clients := h.roomClients(roomID, true, playerID)
	if shouldLogWSMessage(msg.Type) {
		log.Printf("ws broadcast players except type=%s room=%s except_player=%s recipients=%d", msg.Type, roomID, playerID, len(clients))
	}
	for _, c := range clients {
		c.Send(msg)
	}
}

func (h *Hub) SendPlayer(roomID, playerID string, msg protocol.Message) {
	h.mu.RLock()
	c := h.players[roomID][playerID]
	h.mu.RUnlock()
	if c != nil {
		c.Send(msg)
	}
}

func (h *Hub) BroadcastAll(msg protocol.Message) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		c.Send(msg)
	}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
	log.Printf("ws client connected type=%s remote=%s", c.clientType, c.conn.RemoteAddr())
}

func (h *Hub) unregister(c *Client) {
	c.closeOnce.Do(func() {
		h.mu.Lock()
		roomID := c.roomID
		playerID := c.playerID
		clientType := c.clientType
		delete(h.clients, c)
		if c.roomID != "" {
			if set := h.rooms[c.roomID]; set != nil {
				delete(set, c)
				if len(set) == 0 {
					delete(h.rooms, c.roomID)
				}
			}
			if c.playerID != "" {
				if pm := h.players[c.roomID]; pm != nil && pm[c.playerID] == c {
					delete(pm, c.playerID)
					if len(pm) == 0 {
						delete(h.players, c.roomID)
					}
				}
			}
		}
		h.mu.Unlock()
		if clientType == "player" && roomID != "" && playerID != "" {
			h.mgr.RemovePlayer(roomID, playerID)
		}
		close(c.done)
		_ = c.conn.Close()
		log.Printf("ws client disconnected type=%s player=%s room=%s", c.clientType, c.playerID, c.roomID)
	})
}

func (h *Hub) bindRoom(c *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.roomID == roomID {
		return
	}
	if c.roomID != "" {
		if set := h.rooms[c.roomID]; set != nil {
			delete(set, c)
		}
	}
	c.roomID = roomID
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = map[*Client]struct{}{}
	}
	h.rooms[roomID][c] = struct{}{}
}

func (h *Hub) bindPlayer(c *Client, roomID, playerID string) {
	oldRoomID := c.roomID
	oldPlayerID := c.playerID
	h.bindRoom(c, roomID)
	h.mu.Lock()
	defer h.mu.Unlock()
	if oldPlayerID != "" && oldPlayerID != playerID {
		if pm := h.players[oldRoomID]; pm != nil && pm[oldPlayerID] == c {
			delete(pm, oldPlayerID)
		}
	}
	c.playerID = playerID
	if h.players[roomID] == nil {
		h.players[roomID] = map[string]*Client{}
	}
	h.players[roomID][playerID] = c
}

func (h *Hub) roomClients(roomID string, playersOnly bool, exceptPlayerID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	set := h.rooms[roomID]
	out := make([]*Client, 0, len(set))
	for c := range set {
		if playersOnly && c.clientType != "player" {
			continue
		}
		if exceptPlayerID != "" && c.playerID == exceptPlayerID {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (c *Client) Send(msg protocol.Message) {
	select {
	case <-c.done:
		return
	case c.send <- msg:
		if shouldLogWSMessage(msg.Type) {
			log.Printf("ws enqueue type=%s room=%s to_player=%s client_type=%s", msg.Type, msg.RoomID, c.playerID, c.clientType)
		}
	default:
		log.Printf("drop ws message type=%s room=%s player=%s: send queue full", msg.Type, c.roomID, c.playerID)
	}
}

func (c *Client) writeLoop() {
	defer c.hub.unregister(c)
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.send:
			if err := c.conn.WriteJSON(msg); err != nil {
				log.Printf("ws write failed type=%s room=%s player=%s remote=%s err=%v", msg.Type, msg.RoomID, c.playerID, c.conn.RemoteAddr(), err)
				return
			}
			if shouldLogWSMessage(msg.Type) {
				log.Printf("ws sent type=%s room=%s to_player=%s client_type=%s remote=%s", msg.Type, msg.RoomID, c.playerID, c.clientType, c.conn.RemoteAddr())
			}
		}
	}
}

func (c *Client) readLoop() {
	defer c.hub.unregister(c)
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("ws read closed client_type=%s player=%s room=%s remote=%s err=%v", c.clientType, c.playerID, c.roomID, c.conn.RemoteAddr(), err)
			return
		}
		var msg protocol.IncomingMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("ws recv invalid json client_type=%s player=%s room=%s remote=%s bytes=%d err=%v", c.clientType, c.playerID, c.roomID, c.conn.RemoteAddr(), len(data), err)
			c.sendError("", "invalid_json", "WebSocket 消息不是合法 JSON", nil)
			continue
		}
		if shouldLogWSMessage(msg.Type) {
			log.Printf("ws recv type=%s seq=%d msg_room=%s bound_room=%s player=%s client_type=%s remote=%s payload_bytes=%d", msg.Type, msg.Seq, msg.RoomID, c.roomID, c.playerID, c.clientType, c.conn.RemoteAddr(), len(msg.Payload))
		}
		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(msg protocol.IncomingMessage) {
	switch msg.Type {
	case protocol.TypeCreateRoom:
		c.handleCreateRoom(msg)
	case protocol.TypeStartGame:
		c.handleStartGame(msg)
	case protocol.TypeStopGame:
		c.handleStopGame(msg)
	case protocol.TypeSetBounds:
		c.handleSetBounds(msg)
	case protocol.TypePlayerJoin:
		c.handlePlayerJoin(msg)
	case protocol.TypeGrabFlag:
		c.handleGrabFlag(msg)
	case protocol.TypeHeartbeat:
		c.Send(protocol.NewMessage(protocol.TypeHeartbeat, c.hub.mgr.NextSeq(), msg.RoomID, map[string]any{"ok": true}))
	case protocol.TypeSceneMapSave:
		c.handleSceneMapSave(msg)
	case protocol.TypeSceneMapRequest:
		c.handleSceneMapRequest(msg)
	default:
		c.sendError(msg.RoomID, "unknown_type", "未知消息类型", map[string]any{"type": msg.Type})
	}
}

func (c *Client) handleCreateRoom(msg protocol.IncomingMessage) {
	if !c.requireAdmin(msg.RoomID) {
		return
	}
	var cfg protocol.RoomConfig
	if len(msg.Payload) > 0 {
		if err := json.Unmarshal(msg.Payload, &cfg); err != nil {
			c.sendError(msg.RoomID, "invalid_payload", "创建房间 payload 格式错误", err.Error())
			return
		}
	}
	r, err := c.hub.mgr.CreateRoom(cfg)
	if err != nil {
		c.sendCodedError(msg.RoomID, err)
		return
	}
	c.hub.bindRoom(c, r.ID)
	payload := protocol.RoomInfo{RoomID: r.ID, JoinCode: r.JoinCode, Status: string(r.Status)}
	c.Send(protocol.NewMessage(protocol.TypeRoomCreated, c.hub.mgr.NextSeq(), r.ID, payload))
}

func (c *Client) handleStartGame(msg protocol.IncomingMessage) {
	if !c.requireAdmin(msg.RoomID) {
		return
	}
	roomID := c.roomIDOr(msg.RoomID)
	if roomID == "" {
		c.sendError("", "room_not_found", "room_id 不能为空", nil)
		return
	}
	if err := c.hub.mgr.StartGame(roomID); err != nil {
		c.sendCodedError(roomID, err)
	}
}

func (c *Client) handleStopGame(msg protocol.IncomingMessage) {
	if !c.requireAdmin(msg.RoomID) {
		return
	}
	roomID := c.roomIDOr(msg.RoomID)
	var payload protocol.StopGamePayload
	_ = json.Unmarshal(msg.Payload, &payload)
	if err := c.hub.mgr.StopGame(roomID, payload.Reason); err != nil {
		c.sendCodedError(roomID, err)
	}
}

func (c *Client) handleSetBounds(msg protocol.IncomingMessage) {
	if !c.requireAdmin(msg.RoomID) {
		return
	}
	roomID := c.roomIDOr(msg.RoomID)
	var payload protocol.SetBoundsPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError(roomID, "invalid_payload", "设置边界 payload 格式错误", err.Error())
		return
	}
	if err := c.hub.mgr.SetBounds(roomID, payload.Bounds); err != nil {
		c.sendCodedError(roomID, err)
		return
	}
	c.hub.Broadcast(roomID, protocol.NewMessage(protocol.TypeSetBounds, c.hub.mgr.NextSeq(), roomID, payload))
}

func (c *Client) handlePlayerJoin(msg protocol.IncomingMessage) {
	var payload protocol.PlayerJoinPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError(msg.RoomID, "invalid_payload", "加入房间 payload 格式错误", err.Error())
		return
	}
	udpAddr := c.udpAddrFromRemote(payload.UDPPort)
	var (
		r   *room.Room
		p   *room.Player
		err error
	)
	if c.roomID != "" && c.playerID != "" && payload.PlayerID != "" && payload.PlayerID != c.playerID {
		log.Printf("player join rebind requested room=%s old_player=%s new_player=%s display=%s udp_port=%d remote=%s", c.roomID, c.playerID, payload.PlayerID, payload.DisplayName, payload.UDPPort, c.conn.RemoteAddr())
		r, p, err = c.hub.mgr.RebindPlayer(c.roomID, c.playerID, payload.PlayerID, payload.DisplayName, payload.DeviceType, udpAddr)
	} else {
		log.Printf("player join requested join_code=%s player=%s display=%s udp_port=%d bound_room=%s remote=%s", payload.JoinCode, payload.PlayerID, payload.DisplayName, payload.UDPPort, c.roomID, c.conn.RemoteAddr())
		r, p, err = c.hub.mgr.JoinRoom(payload.JoinCode, payload.PlayerID, payload.DisplayName, payload.DeviceType, udpAddr)
	}
	if err != nil {
		c.sendCodedError(msg.RoomID, err)
		return
	}
	c.hub.bindPlayer(c, r.ID, p.ID)
	joinedPayload, err := c.hub.mgr.PlayerJoinedPayload(r.ID, p.ID)
	if err != nil {
		c.sendCodedError(r.ID, err)
		return
	}
	log.Printf("player joined room=%s player=%s slot=%d count=%v remote=%s", r.ID, p.ID, p.Slot, joinedPayloadValue(joinedPayload, "player_count"), c.conn.RemoteAddr())
	c.hub.Broadcast(r.ID, protocol.NewMessage(protocol.TypePlayerJoined, c.hub.mgr.NextSeq(), r.ID, joinedPayload))
	c.sendExistingPlayers(r.ID, p.ID, "player_join")
	if scene, err := c.hub.mgr.SceneMap(r.ID); err == nil && scene != nil {
		log.Printf("scene snapshot after join room=%s to_player=%s version=%d map_id=%s bytes=%d", r.ID, p.ID, scene.Version, scene.MapID, len(scene.Map))
		c.Send(protocol.NewMessage(protocol.TypeSceneMapSnapshot, c.hub.mgr.NextSeq(), r.ID, scene))
	} else if err != nil {
		log.Printf("scene snapshot after join failed room=%s to_player=%s err=%v", r.ID, p.ID, err)
	} else {
		log.Printf("scene snapshot after join skipped: no scene map room=%s to_player=%s", r.ID, p.ID)
	}
}

func (c *Client) autoJoinDefault(r *http.Request) {
	defaultRoom, err := c.hub.mgr.EnsureDefaultRoom()
	if err != nil {
		c.sendCodedError("", err)
		return
	}
	c.hub.bindRoom(c, defaultRoom.ID)
	log.Printf("auto default room ready room=%s join_code=%s status=%s remote=%s", defaultRoom.ID, defaultRoom.JoinCode, defaultRoom.Status, c.conn.RemoteAddr())
	c.Send(protocol.NewMessage(protocol.TypeRoomCreated, c.hub.mgr.NextSeq(), defaultRoom.ID, protocol.RoomInfo{
		RoomID:   defaultRoom.ID,
		JoinCode: defaultRoom.JoinCode,
		Status:   string(defaultRoom.Status),
	}))

	playerID := strings.TrimSpace(r.Header.Get("X-Player-ID"))
	if playerID == "" {
		playerID = strings.TrimSpace(r.URL.Query().Get("player_id"))
	}
	displayName := strings.TrimSpace(r.Header.Get("X-Display-Name"))
	if displayName == "" {
		displayName = strings.TrimSpace(r.URL.Query().Get("display_name"))
	}
	deviceType := strings.TrimSpace(r.Header.Get("X-Device-Type"))
	if deviceType == "" {
		deviceType = strings.TrimSpace(r.URL.Query().Get("device_type"))
	}
	defaultRoom, p, err := c.hub.mgr.AutoJoinDefaultPlayer(playerID, displayName, deviceType, nil)
	if err != nil {
		roomID := ""
		if defaultRoom != nil {
			roomID = defaultRoom.ID
		}
		c.sendCodedError(roomID, err)
		return
	}
	c.hub.bindPlayer(c, defaultRoom.ID, p.ID)
	log.Printf("auto default player joined room=%s player=%s slot=%d display=%s remote=%s", defaultRoom.ID, p.ID, p.Slot, p.DisplayName, c.conn.RemoteAddr())
	joinedPayload, err := c.hub.mgr.PlayerJoinedPayload(defaultRoom.ID, p.ID)
	if err != nil {
		c.sendCodedError(defaultRoom.ID, err)
		return
	}
	c.hub.Broadcast(defaultRoom.ID, protocol.NewMessage(protocol.TypePlayerJoined, c.hub.mgr.NextSeq(), defaultRoom.ID, joinedPayload))
	c.sendExistingPlayers(defaultRoom.ID, p.ID, "auto_join")
	if scene, err := c.hub.mgr.SceneMap(defaultRoom.ID); err == nil && scene != nil {
		log.Printf("scene snapshot after auto join room=%s to_player=%s version=%d map_id=%s bytes=%d", defaultRoom.ID, p.ID, scene.Version, scene.MapID, len(scene.Map))
		c.Send(protocol.NewMessage(protocol.TypeSceneMapSnapshot, c.hub.mgr.NextSeq(), defaultRoom.ID, scene))
	} else if err != nil {
		log.Printf("scene snapshot after auto join failed room=%s to_player=%s err=%v", defaultRoom.ID, p.ID, err)
	} else {
		log.Printf("scene snapshot after auto join skipped: no scene map room=%s to_player=%s", defaultRoom.ID, p.ID)
	}
}

func (c *Client) sendExistingPlayers(roomID, newPlayerID, reason string) {
	snapshot, err := c.hub.mgr.Snapshot(roomID)
	if err != nil {
		log.Printf("roster sync failed room=%s to_player=%s reason=%s err=%v", roomID, newPlayerID, reason, err)
		return
	}
	sent := 0
	ready := len(snapshot.Players) >= 2
	for _, p := range snapshot.Players {
		if p.ID == newPlayerID {
			continue
		}
		payload := map[string]any{
			"player_id":    p.ID,
			"display_name": p.DisplayName,
			"slot":         p.Slot,
			"spawn_pos":    p.SpawnPos,
			"player_count": len(snapshot.Players),
			"ready":        ready,
		}
		c.Send(protocol.NewMessage(protocol.TypePlayerJoined, c.hub.mgr.NextSeq(), roomID, payload))
		sent++
	}
	log.Printf("roster sync room=%s to_player=%s reason=%s existing_sent=%d total_players=%d", roomID, newPlayerID, reason, sent, len(snapshot.Players))
}

func (c *Client) handleGrabFlag(msg protocol.IncomingMessage) {
	var payload protocol.GrabFlagPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError(msg.RoomID, "invalid_payload", "抓旗 payload 格式错误", err.Error())
		return
	}
	roomID := c.playerBoundRoomID(msg.RoomID, msg.Type)
	if roomID == "" {
		c.sendError("", "room_not_found", "room_id 不能为空", nil)
		return
	}
	if payload.PlayerID == "" {
		payload.PlayerID = c.playerID
	} else if c.clientType == "player" && c.playerID != "" && payload.PlayerID != c.playerID {
		log.Printf("player payload player_id overridden type=%s payload_player=%s bound_player=%s room=%s remote=%s", msg.Type, payload.PlayerID, c.playerID, roomID, c.conn.RemoteAddr())
		payload.PlayerID = c.playerID
	}
	if err := c.hub.mgr.GrabFlag(roomID, payload.PlayerID, payload.FlagID, payload.Pos); err != nil {
		c.sendCodedError(roomID, err)
	}
}

func (c *Client) handleSceneMapSave(msg protocol.IncomingMessage) {
	roomID := c.playerBoundRoomID(msg.RoomID, msg.Type)
	if roomID == "" {
		c.sendError("", "room_not_found", "room_id 不能为空", nil)
		return
	}
	var payload room.SceneMapSavePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError(roomID, "invalid_payload", "场景地图 payload 格式错误", err.Error())
		return
	}
	if payload.PlayerID == "" {
		payload.PlayerID = c.playerID
	} else if c.clientType == "player" && c.playerID != "" && payload.PlayerID != c.playerID {
		log.Printf("player payload player_id overridden type=%s payload_player=%s bound_player=%s room=%s remote=%s", msg.Type, payload.PlayerID, c.playerID, roomID, c.conn.RemoteAddr())
		payload.PlayerID = c.playerID
	}
	log.Printf("scene save requested room=%s player=%s map_id=%s base_version=%d force=%v schema=%s anchor=%s coord=%s bytes=%d remote=%s", roomID, payload.PlayerID, payload.MapID, payload.BaseVersion, payload.Force, payload.SchemaVersion, payload.AnchorID, payload.CoordinateSpace, len(payload.Map), c.conn.RemoteAddr())
	saved, err := c.hub.mgr.SaveSceneMap(roomID, payload)
	if err != nil {
		log.Printf("scene save failed room=%s player=%s err=%v", roomID, payload.PlayerID, err)
		c.sendCodedError(roomID, err)
		return
	}
	log.Printf("scene saved room=%s player=%s map_id=%s version=%d bytes=%d updated_ts=%d", roomID, payload.PlayerID, saved.MapID, saved.Version, len(saved.Map), saved.UpdatedTs)
	c.Send(protocol.NewMessage(protocol.TypeSceneMapSaved, c.hub.mgr.NextSeq(), roomID, saved))
	c.hub.BroadcastPlayersExcept(roomID, payload.PlayerID, protocol.NewMessage(protocol.TypeSceneMapUpdate, c.hub.mgr.NextSeq(), roomID, saved))
}

func (c *Client) handleSceneMapRequest(msg protocol.IncomingMessage) {
	roomID := c.playerBoundRoomID(msg.RoomID, msg.Type)
	if roomID == "" {
		c.sendError("", "room_not_found", "room_id 不能为空", nil)
		return
	}
	scene, err := c.hub.mgr.SceneMap(roomID)
	if err != nil {
		log.Printf("scene request failed room=%s player=%s err=%v", roomID, c.playerID, err)
		c.sendCodedError(roomID, err)
		return
	}
	if scene == nil {
		log.Printf("scene request result: no scene map room=%s player=%s remote=%s", roomID, c.playerID, c.conn.RemoteAddr())
	} else {
		log.Printf("scene request result room=%s player=%s version=%d map_id=%s bytes=%d remote=%s", roomID, c.playerID, scene.Version, scene.MapID, len(scene.Map), c.conn.RemoteAddr())
	}
	c.Send(protocol.NewMessage(protocol.TypeSceneMapSnapshot, c.hub.mgr.NextSeq(), roomID, scene))
}

func (c *Client) requireAdmin(roomID string) bool {
	if c.clientType == "admin" {
		return true
	}
	c.sendError(roomID, "unauthorized", "该消息类型仅 PC 管理端可用", nil)
	return false
}

func (c *Client) roomIDOr(roomID string) string {
	if roomID != "" {
		return roomID
	}
	return c.roomID
}

func (c *Client) playerBoundRoomID(msgRoomID, msgType string) string {
	if c.clientType == "player" && c.roomID != "" {
		if msgRoomID != "" && msgRoomID != c.roomID {
			log.Printf("player message room overridden type=%s msg_room=%s bound_room=%s player=%s remote=%s", msgType, msgRoomID, c.roomID, c.playerID, c.conn.RemoteAddr())
		}
		return c.roomID
	}
	return c.roomIDOr(msgRoomID)
}

func (c *Client) udpAddrFromRemote(port int) *net.UDPAddr {
	if port <= 0 {
		return nil
	}
	host, _, err := net.SplitHostPort(c.conn.RemoteAddr().String())
	if err != nil {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil
	}
	return &net.UDPAddr{IP: ip, Port: port}
}

func (c *Client) sendCodedError(roomID string, err error) {
	if ce, ok := room.AsCodedError(err); ok {
		log.Printf("ws coded error room=%s player=%s code=%s message=%s detail=%v", roomID, c.playerID, ce.Code, ce.Message, ce.Detail)
		c.sendError(roomID, ce.Code, ce.Message, ce.Detail)
		return
	}
	log.Printf("ws internal error room=%s player=%s err=%v", roomID, c.playerID, err)
	c.sendError(roomID, "internal_error", err.Error(), nil)
}

func (c *Client) sendError(roomID, code, message string, detail any) {
	log.Printf("ws send error room=%s player=%s code=%s message=%s detail=%v", roomID, c.playerID, code, message, detail)
	c.Send(protocol.NewMessage(protocol.TypeError, c.hub.mgr.NextSeq(), roomID, protocol.ErrorPayload{
		Code:    code,
		Message: message,
		Detail:  detail,
	}))
}

func shouldLogWSMessage(msgType string) bool {
	switch msgType {
	case protocol.TypeHeartbeat, protocol.TypePlayerState:
		return false
	default:
		return true
	}
}

func joinedPayloadValue(payload any, key string) any {
	m, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	return m[key]
}
