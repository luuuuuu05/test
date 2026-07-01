using System;
using System.Collections.Generic;
using Newtonsoft.Json;
using Newtonsoft.Json.Converters;

[Serializable]
public class BaseMessage
{
    public string type;
    public uint seq;
    public long ts;
    public string room_id;
    public string payload;
}

[Serializable]
public class RoomCreatedPayload
{
    public string room_id;
    public string join_code;
    public string status;
}

[Serializable]
public class PlayerJoinPayload
{
    public string player_id;
    public string join_code;
    public string display_name;
    public string device_type;
    public int udp_port;
}

[Serializable]
public class PlayerJoinedPayload
{
    public string player_id;
    public string display_name;
    public int slot;
    public Vector3Json spawn_pos;
    public int player_count;
    public bool ready;
}

[Serializable]
public class CountdownPayload
{
    public int count;
}

[Serializable]
public class FlagSpawnPayload
{
    public string flag_id;
    public string flag_type;
    public int score;
    public Vector3Json pos;
    public string flag_point_id;
    public float lifetime;
}

[Serializable]
public class FlagRemovePayload
{
    public string flag_id;
    public string reason;
    public string grabbed_by;
}

[Serializable]
public class GrabFlagPayload
{
    public string player_id;
    public string flag_id;
    public Vector3Json pos;
}

[Serializable]
public class ScoreUpdatePayload
{
    public string player_id;
    public int delta;
    public int total;
    public bool double_active;
    public List<PlayerScoreEntry> scoreboard;
}

[Serializable]
public class PlayerScoreEntry
{
    public string player_id;
    public string display_name;
    public int score;
}

[Serializable]
public class BuffPayload
{
    public string player_id;
    public string buff_type;
    public float duration;
    public long end_ts;
}

[Serializable]
public class BuffEndPayload
{
    public string player_id;
    public string buff_type;
    public string reason;
}

[Serializable]
public class GameEndPayload
{
    public string reason;
    public int duration_actual;
    public List<GameResultEntry> final_scores;
}

[Serializable]
public class GameResultEntry
{
    public int rank;
    public string player_id;
    public string display_name;
    public int score;
    public int flags_grabbed;
    public string award;
}

[Serializable]
public class ErrorPayload
{
    public string code;
    public string message;
    public object detail;
}

[Serializable]
public class PlayerStatePayload
{
    public string player_id;
    public Vector3Json pos;
    public float rot_y;
    public float head_pitch;
}

[Serializable]
public class Vector3Json
{
    public float x;
    public float y;
    public float z;
}

[Serializable]
public class CreateRoomPayload
{
    public string room_name;
    public int game_duration;
    public int max_flags;
    public int min_flags;
    public float respawn_delay;
    public int max_double_items;
    public BoundsJson bounds;
    public List<FlagPointJson> flag_points;
}

[Serializable]
public class BoundsJson
{
    public float x_min;
    public float x_max;
    public float z_min;
    public float z_max;
}

[Serializable]
public class FlagPointJson
{
    public string id;
    public float x;
    public float y;
    public float z;
}

public static class MessageTypes
{
    public const string C_CREATE_ROOM = "c_create_room";
    public const string S_ROOM_CREATED = "s_room_created";
    public const string C_START_GAME = "c_start_game";
    public const string C_STOP_GAME = "c_stop_game";
    public const string C_PLAYER_JOIN = "c_player_join";
    public const string S_PLAYER_JOINED = "s_player_joined";
    public const string S_GAME_COUNTDOWN = "s_game_countdown";
    public const string S_GAME_START = "s_game_start";
    public const string S_GAME_END = "s_game_end";
    public const string C_GRAB_FLAG = "c_grab_flag";
    public const string S_FLAG_SPAWN = "s_flag_spawn";
    public const string S_FLAG_REMOVE = "s_flag_remove";
    public const string S_SCORE_UPDATE = "s_score_update";
    public const string S_BUFF_START = "s_buff_start";
    public const string S_BUFF_END = "s_buff_end";
    public const string S_PLAYER_STATE = "s_player_state";
    public const string S_ERROR = "s_error";
    public const string C_HEARTBEAT = "c_heartbeat";
    public const string C_SCENE_MAP_SAVE = "c_scene_map_save";
    public const string S_SCENE_MAP_SAVED = "s_scene_map_saved";
    public const string S_SCENE_MAP_UPDATE = "s_scene_map_update";
    public const string C_SCENE_MAP_REQUEST = "c_scene_map_request";
    public const string S_SCENE_MAP_SNAPSHOT = "s_scene_map_snapshot";
}