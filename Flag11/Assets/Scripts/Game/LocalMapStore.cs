using System;
using System.Collections.Generic;
using System.IO;
using UnityEngine;
using Newtonsoft.Json;

public static class LocalMapStore
{
    private static string MapsDir
    {
        get
        {
            string dir = Path.Combine(Application.persistentDataPath, "maps");
            if (!Directory.Exists(dir)) Directory.CreateDirectory(dir);
            return dir;
        }
    }

    [Serializable]
    public class MapEntry
    {
        public string mapName;
        public long savedAt;
        public int objectCount;
        public List<SceneObject> objects;
    }

    public static void Save(string mapName, List<SceneObject> objects)
    {
        var entry = new MapEntry
        {
            mapName = mapName,
            savedAt = DateTimeOffset.UtcNow.ToUnixTimeSeconds(),
            objectCount = objects.Count,
            objects = objects
        };

        string json = JsonConvert.SerializeObject(entry, Formatting.Indented);
        string path = Path.Combine(MapsDir, SanitizeFileName(mapName) + ".json");
        File.WriteAllText(path, json);
        Debug.Log($"[LocalMap] Saved '{mapName}' ({objects.Count} objects) to {path}");
    }

    public static MapEntry Load(string mapName)
    {
        string path = Path.Combine(MapsDir, SanitizeFileName(mapName) + ".json");
        if (!File.Exists(path)) return null;

        string json = File.ReadAllText(path);
        return JsonConvert.DeserializeObject<MapEntry>(json);
    }

    public static List<string> ListAll()
    {
        var names = new List<string>();
        if (!Directory.Exists(MapsDir)) return names;

        foreach (string file in Directory.GetFiles(MapsDir, "*.json"))
        {
            names.Add(Path.GetFileNameWithoutExtension(file));
        }
        return names;
    }

    public static void Delete(string mapName)
    {
        string path = Path.Combine(MapsDir, SanitizeFileName(mapName) + ".json");
        if (File.Exists(path)) File.Delete(path);
        Debug.Log($"[LocalMap] Deleted '{mapName}'");
    }

    private static string SanitizeFileName(string name)
    {
        foreach (char c in Path.GetInvalidFileNameChars())
            name = name.Replace(c, '_');
        return string.IsNullOrEmpty(name) ? "unnamed" : name;
    }
}