using UnityEngine;

public static class MapLoaderHelper
{
    public static void LoadMapIntoScene(SceneObjectRegistry objectRegistry)
    {
        string mapName = PlayerPrefs.GetString("SelectedMap", "");
        if (string.IsNullOrEmpty(mapName)) return;

        var entry = LocalMapStore.Load(mapName);
        if (entry == null || entry.objects == null) return;

        foreach (var obj in entry.objects)
        {
            GameObject go = null;
            if (objectRegistry != null && obj.prefab != null)
            {
                var prefab = objectRegistry.GetPrefab(obj.prefab);
                if (prefab != null) go = Object.Instantiate(prefab);
            }

            if (go == null)
            {
                PrimitiveType prim = PrimitiveType.Cube;
                Color color = Color.gray;
                if (obj.prefab != null)
                {
                    string p = obj.prefab.ToLower();
                    if (p.Contains("pillar") || p.Contains("barrel")) { prim = PrimitiveType.Cylinder; color = new Color(0.5f, 0.3f, 0.1f); }
                    else if (p.Contains("sphere")) { prim = PrimitiveType.Sphere; color = new Color(0.2f, 0.6f, 0.2f); }
                    else if (p.Contains("box")) color = new Color(0.6f, 0.3f, 0.1f);
                    else if (p.Contains("wall")) color = new Color(0.3f, 0.3f, 0.4f);
                }
                go = GameObject.CreatePrimitive(prim);
                var r = go.GetComponent<Renderer>();
                if (r != null) r.material.color = color;
            }

            go.transform.position = new Vector3(obj.pos.x, 0.02f, obj.pos.z);
            go.transform.eulerAngles = new Vector3(obj.rot.x, obj.rot.y, obj.rot.z);
            go.transform.localScale = new Vector3(obj.scale.x, obj.scale.y, obj.scale.z);
        }
    }
}
