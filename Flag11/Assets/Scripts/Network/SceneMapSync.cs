using System;
using UnityEngine;
using Newtonsoft.Json;

public class SceneMapSync : MonoBehaviour
{
    private NetworkManager _network;
    private int _currentVersion = 0;

    public SceneObjectRegistry objectRegistry;

    void Start()
    {
        _network = GetComponent<NetworkManager>();
        if (_network != null)
        {
            _network.OnServerMessage += HandleMessage;
        }
    }

    private void HandleMessage(string type, Newtonsoft.Json.Linq.JObject payload)
    {
        switch (type)
        {
            case MessageTypes.S_SCENE_MAP_SAVED:
                var saved = payload.ToObject<SceneMapAckPayload>();
                if (saved != null) _currentVersion = saved.version;
                Debug.Log($"[SceneMap] Saved ack, version={_currentVersion}");
                break;

            case MessageTypes.S_SCENE_MAP_UPDATE:
                var update = payload.ToObject<SceneMapUpdatePayload>();
                if (update != null)
                {
                    _currentVersion = update.version;
                    ApplyMap(update.map);
                }
                break;

            case MessageTypes.S_SCENE_MAP_SNAPSHOT:
                var snapshot = payload.ToObject<SceneMapUpdatePayload>();
                if (snapshot != null)
                {
                    _currentVersion = snapshot.version;
                    ApplyMap(snapshot.map);
                }
                break;
        }
    }

    public void SaveMap(SceneMap map, bool force = false)
    {
        var payload = new SceneMapPayload
        {
            player_id = GameContext.Instance.PlayerID,
            map_id = "default",
            base_version = _currentVersion,
            force = force,
            schema_version = "mrflag.scene.v1",
            anchor_id = "ANCHOR_CENTER_001",
            coordinate_space = "shared_anchor",
            map = map
        };

        _network.SendToServer(MessageTypes.C_SCENE_MAP_SAVE, payload);
    }

    public void RequestMap()
    {
        _network.SendToServer(MessageTypes.C_SCENE_MAP_REQUEST, new { });
    }

    private void ApplyMap(SceneMap map)
    {
        if (map?.objects == null) return;

        foreach (var obj in map.objects)
        {
            var pos = new Vector3(obj.pos.x, obj.pos.y, obj.pos.z);
            var rot = Quaternion.Euler(obj.rot.x, obj.rot.y, obj.rot.z);

            var existing = GameObject.Find(obj.id);
            if (existing != null)
            {
                existing.transform.position = pos;
                existing.transform.rotation = rot;
                if (obj.scale != null)
                    existing.transform.localScale = new Vector3(obj.scale.x, obj.scale.y, obj.scale.z);
            }
            else
            {
                GameObject go = null;
                if (objectRegistry != null && obj.prefab != null)
                {
                    var prefab = objectRegistry.GetPrefab(obj.prefab);
                    if (prefab != null) go = Instantiate(prefab);
                }

                if (go == null)
                    go = new GameObject(obj.id);

                go.name = obj.id;
                go.transform.position = pos;
                go.transform.rotation = rot;
                if (obj.scale != null)
                    go.transform.localScale = new Vector3(obj.scale.x, obj.scale.y, obj.scale.z);
            }
        }

        Debug.Log($"[SceneMap] Applied {map.objects.Count} objects");
    }

    void OnDestroy()
    {
        if (_network != null)
            _network.OnServerMessage -= HandleMessage;
    }
}