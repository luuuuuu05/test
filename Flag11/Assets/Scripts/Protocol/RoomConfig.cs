using System;
using System.Collections.Generic;

[Serializable]
public class RoomConfig
{
    public string room_name;
    public int game_duration = 180;
    public int max_flags = 5;
    public int min_flags = 4;
    public float respawn_delay = 2.0f;
    public int max_double_items = 1;
    public BoundsJson bounds;
    public List<FlagPointJson> flag_points;
}

[Serializable]
public class SceneMapPayload
{
    public string player_id;
    public string map_id;
    public int base_version;
    public bool force;
    public string schema_version;
    public string anchor_id;
    public string coordinate_space;
    public SceneMap map;
}

[Serializable]
public class SceneMap
{
    public List<SceneObject> objects;
}

[Serializable]
public class SceneObject
{
    public string id;
    public string prefab;
    public Vector3Json pos;
    public Vector3Json rot;
    public Vector3Json scale;
    public Dictionary<string, string> props;
}

[Serializable]
public class SceneMapAckPayload
{
    public string map_id;
    public int version;
    public string schema_version;
    public string anchor_id;
    public string coordinate_space;
    public string updated_by;
    public long updated_ts;
    public SceneMap map;
}

[Serializable]
public class SceneMapUpdatePayload
{
    public string map_id;
    public int version;
    public string updated_by;
    public long updated_ts;
    public SceneMap map;
}