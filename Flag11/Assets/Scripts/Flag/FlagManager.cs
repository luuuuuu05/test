using System.Collections.Generic;
using UnityEngine;

public class FlagManager : MonoBehaviour
{
    public GameObject flagPrefab;
    private Dictionary<string, FlagObject> _flags = new();

    public IReadOnlyDictionary<string, FlagObject> ActiveFlags => _flags;

    public void SpawnFlag(FlagSpawnPayload data)
    {
        if (_flags.ContainsKey(data.flag_id))
        {
            Debug.LogWarning($"[Flag] Flag {data.flag_id} already exists, skipping");
            return;
        }

        GameObject obj;
        if (flagPrefab != null)
        {
            obj = Instantiate(flagPrefab);
        }
        else
        {
            obj = new GameObject();
        }

        var flagComp = obj.GetComponent<FlagObject>();
        if (flagComp == null) flagComp = obj.AddComponent<FlagObject>();

        flagComp.Initialize(data);
        _flags[data.flag_id] = flagComp;

        Debug.Log($"[Flag] Spawned {data.flag_id} ({data.flag_type}, {data.score}pts)");
    }

    public void RemoveFlag(string flagId)
    {
        if (_flags.TryGetValue(flagId, out var flag))
        {
            Debug.Log($"[Flag] Removed {flagId}");
            Destroy(flag.gameObject);
            _flags.Remove(flagId);
        }
    }

    public void ClearAll()
    {
        foreach (var kv in _flags)
        {
            if (kv.Value != null) Destroy(kv.Value.gameObject);
        }
        _flags.Clear();
    }

    public FlagObject GetFlag(string flagId)
    {
        _flags.TryGetValue(flagId, out var flag);
        return flag;
    }
}