using System;
using System.Collections.Generic;
using UnityEngine;

[CreateAssetMenu(fileName = "SceneObjectRegistry", menuName = "Flag/Scene Object Registry")]
public class SceneObjectRegistry : ScriptableObject
{
    [Serializable]
    public class Entry
    {
        public string displayName;
        public GameObject prefab;
        public Sprite icon;
    }

    public List<Entry> entries = new();

    public int Count => entries.Count;

    public Entry GetEntry(int index)
    {
        if (index < 0 || index >= entries.Count) return null;
        return entries[index];
    }

    public GameObject GetPrefab(string prefabName)
    {
        foreach (var e in entries)
        {
            if (e.prefab != null && e.prefab.name == prefabName)
                return e.prefab;
        }
        return null;
    }

    public int FindIndex(string prefabName)
    {
        for (int i = 0; i < entries.Count; i++)
        {
            if (entries[i].prefab != null && entries[i].prefab.name == prefabName)
                return i;
        }
        return -1;
    }
}