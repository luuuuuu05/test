# MR Flag Battle Server

Go 后端实现，按 `MR_FlagBattle_TechDoc_v1.0.docx` 的 HTTP / WebSocket / UDP 协议落地。

## 启动

```powershell
$env:GOCACHE='E:\Program Files\projects\MRFlag\.cache\go-build'
go run ./cmd/server -config config.yaml
```

默认端口：

- HTTP REST: `:8080`
- WebSocket: `:8081/ws`
- UDP position relay: `:9090`

## REST

| Method | Path | 说明 |
| --- | --- | --- |
| `GET` | `/health` | 健康检查 |
| `POST` | `/api/rooms` | 创建房间，Body 为 `RoomConfig` |
| `GET` | `/api/rooms/:id` | 房间快照 |
| `POST` | `/api/rooms/:id/start` | 开始游戏 |
| `POST` | `/api/rooms/:id/stop` | 终止游戏 |
| `GET` | `/api/rooms/:id/scores` | 当前分数 |
| `GET` | `/api/rooms/:id/flags` | 当前旗帜 |
| `GET` | `/api/rooms/:id/scene-map` | 当前场景地图 |
| `PUT` | `/api/rooms/:id/scene-map` | 管理端写入场景地图 |

## WebSocket

连接：

```http
GET ws://SERVER_IP:8081/ws
X-Client-Type: admin | player
X-Room-ID: ROOM_ABC123
```

开发简化模式：`player` 客户端不带 `X-Room-ID`、不发 `c_create_room` 也能用。服务端会自动创建/复用 `ROOM_DEFAULT`，邀请码固定为 `DEFAULT`，连接后立即下发 `s_room_created` 并自动广播 `s_player_joined`。如果 Unity 后续再发 `c_player_join`，服务端会把临时 `AUTO_xxx` 玩家重绑定成 Unity 提供的 `player_id`。

所有 WS 消息遵循：

```json
{
  "type": "string",
  "seq": 1,
  "ts": 1700000000000,
  "room_id": "ROOM_ABC123",
  "payload": {}
}
```

已实现文档里的核心类型：

- `c_create_room` / `s_room_created`
- `c_player_join` / `s_player_joined`
- `c_start_game` / `s_game_countdown` / `s_game_start`
- `c_stop_game` / `s_game_end`
- `c_grab_flag` / `s_flag_remove` / `s_score_update`
- `s_flag_spawn`
- `s_buff_start` / `s_buff_end`
- `c_heartbeat`
- `s_error`

## 场景地图同步协议

Unity 端拖拽摆放完物体后，发 `c_scene_map_save`。服务端保存房间内最新 JSON 地图，版本号自增，给上传者回 ACK，并把完整地图推给同房间其他玩家。

### 上传地图

```json
{
  "type": "c_scene_map_save",
  "seq": 101,
  "ts": 1700000000000,
  "room_id": "ROOM_ABC123",
  "payload": {
    "player_id": "P_001",
    "map_id": "default",
    "base_version": 0,
    "force": false,
    "schema_version": "mrflag.scene.v1",
    "anchor_id": "ANCHOR_CENTER_001",
    "coordinate_space": "shared_anchor",
    "map": {
      "objects": [
        {
          "id": "OBJ_BOX_001",
          "prefab": "CoverBox",
          "pos": { "x": 1.2, "y": 0.0, "z": 2.4 },
          "rot": { "x": 0.0, "y": 90.0, "z": 0.0 },
          "scale": { "x": 1.0, "y": 1.0, "z": 1.0 },
          "props": { "team": "neutral" }
        }
      ]
    }
  }
}
```

### 上传者 ACK

```json
{
  "type": "s_scene_map_saved",
  "seq": 102,
  "ts": 1700000000100,
  "room_id": "ROOM_ABC123",
  "payload": {
    "map_id": "default",
    "version": 1,
    "schema_version": "mrflag.scene.v1",
    "anchor_id": "ANCHOR_CENTER_001",
    "coordinate_space": "shared_anchor",
    "updated_by": "P_001",
    "updated_ts": 1700000000100,
    "map": {}
  }
}
```

### 其他玩家收到同步

```json
{
  "type": "s_scene_map_update",
  "seq": 103,
  "ts": 1700000000100,
  "room_id": "ROOM_ABC123",
  "payload": {
    "map_id": "default",
    "version": 1,
    "updated_by": "P_001",
    "updated_ts": 1700000000100,
    "map": {}
  }
}
```

### 主动请求或新玩家加入自动快照

```json
{
  "type": "c_scene_map_request",
  "seq": 104,
  "ts": 1700000000200,
  "room_id": "ROOM_ABC123",
  "payload": {}
}
```

服务端返回 `s_scene_map_snapshot`，payload 同 `s_scene_map_update`。新玩家 `c_player_join` 成功后，如果房间已有地图，也会自动收到 `s_scene_map_snapshot`。

### 版本冲突

如果 `base_version` 和服务端当前版本不一致且 `force=false`，返回：

```json
{
  "type": "s_error",
  "payload": {
    "code": "scene_map_conflict",
    "message": "场景地图版本冲突",
    "detail": {
      "current_version": 2,
      "base_version": 1
    }
  }
}
```

Unity 端想强制覆盖时把 `force` 设为 `true`。

## UDP MessagePack

上行仍按文档的 10 项数组：

```text
[room_id, player_id, seq, ts, x, y, z, rot_y, head_pitch, flags]
```

下行 relay 为 9 项数组：

```text
[from_player_id, seq, server_ts, x, y, z, rot_y, head_pitch, flags]
```
