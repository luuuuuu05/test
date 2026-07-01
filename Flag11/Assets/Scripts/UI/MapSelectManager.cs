using System.Collections.Generic;
using System.IO;
using UnityEngine;
using UnityEngine.UI;
using UnityEngine.SceneManagement;

public class MapSelectManager : MonoBehaviour
{
    public Transform mapListParent;
    public Button mapItemPrefab;
    public Text titleText;
    public Button backButton, newMapButton;

    private List<GameObject> _createdItems = new();
    private NetworkManager _net;

    void Start()
    {
        Debug.Log("[MapSelect] Start");
        _net = NetworkManager.Instance;

        if (backButton != null) backButton.onClick.AddListener(() => SceneManager.LoadScene("MainMenu"));
        if (newMapButton != null) newMapButton.onClick.AddListener(() => SceneManager.LoadScene("MapEditor"));

        // Listen for remote map selection
        if (_net != null) _net.OnServerMessage += OnServerMsg;

        LoadMapList();
    }

    void LoadMapList()
    {
        if (mapListParent == null || mapItemPrefab == null) return;
        foreach (var go in _createdItems) Destroy(go);
        _createdItems.Clear();

        var maps = LocalMapStore.ListAll();
        Debug.Log("[MapSelect] Found " + maps.Count + " maps");
        foreach (string mapName in maps) CreateMapButton(mapName);
    }

    void CreateMapButton(string mapName)
    {
        var row = new GameObject("Row_" + mapName, typeof(RectTransform), typeof(HorizontalLayoutGroup));
        row.transform.SetParent(mapListParent, false);
        row.GetComponent<HorizontalLayoutGroup>().childForceExpandWidth = true;

        var btnObj = Instantiate(mapItemPrefab, row.transform);
        btnObj.gameObject.SetActive(true);
        var txt = btnObj.GetComponentInChildren<Text>();
        if (txt != null) txt.text = mapName;
        var n = mapName;
        btnObj.onClick.AddListener(() => SelectMap(n));

        // Delete button
        var delBtn = new GameObject("DelBtn", typeof(RectTransform), typeof(UnityEngine.UI.Image), typeof(UnityEngine.UI.Button));
        delBtn.GetComponent<RectTransform>().sizeDelta = new Vector2(50, 50);
        delBtn.GetComponent<UnityEngine.UI.Image>().color = new Color(0.8f, 0.2f, 0.2f);
        delBtn.transform.SetParent(row.transform, false);
        var delTxt = new GameObject("Text", typeof(RectTransform), typeof(UnityEngine.UI.Text));
        delTxt.GetComponent<UnityEngine.UI.Text>().text = "X";
        delTxt.GetComponent<UnityEngine.UI.Text>().fontSize = 24;
        delTxt.GetComponent<UnityEngine.UI.Text>().color = Color.white;
        delTxt.GetComponent<UnityEngine.UI.Text>().alignment = TextAnchor.MiddleCenter;
        delTxt.transform.SetParent(delBtn.transform, false);
        delTxt.GetComponent<RectTransform>().anchorMin = Vector2.zero;
        delTxt.GetComponent<RectTransform>().anchorMax = Vector2.one;
        delTxt.GetComponent<RectTransform>().sizeDelta = Vector2.zero;
        delBtn.GetComponent<UnityEngine.UI.Button>().onClick.AddListener(() => DeleteMap(n));

        _createdItems.Add(row);
    }

    void SelectMap(string mapName)
    {
        Debug.Log("[MapSelect] Selected: " + mapName);
        PlayerPrefs.SetString("SelectedMap", mapName);

        var ctx = GameContext.Instance;
        bool isMulti = ctx != null && !string.IsNullOrEmpty(ctx.serverIP);

        if (isMulti && _net != null)
        {
            // Use the server's scene-map protocol
            var entry = LocalMapStore.Load(mapName);
            if (entry != null)
            {
                _net.SendToServer("c_scene_map_save", new SceneMapPayload
                {
                    player_id = ctx.PlayerID,
                    map_id = mapName,
                    base_version = 0,
                    force = true,
                    schema_version = "mrflag.scene.v1",
                    anchor_id = "",
                    coordinate_space = "world",
                    map = new SceneMap { objects = entry.objects }
                });
            }
        }

        string targetScene = isMulti ? "MultiGame" : "MRGame";
        SceneManager.LoadScene(targetScene);
    }

    string LoadMapJson(string mapName)
    {
        string path = Application.persistentDataPath + "/maps/" + mapName + ".json";
        if (!System.IO.File.Exists(path)) return null;
        return System.IO.File.ReadAllText(path);
    }

    void OnServerMsg(string type, Newtonsoft.Json.Linq.JObject payload)
    {
        if (type == "s_scene_map_saved" || type == "s_scene_map_snapshot" || type == "s_scene_map_update")
        {
            string mapId = payload["map_id"]?.ToString() ?? "default";
            var mapObj = payload["map"] as Newtonsoft.Json.Linq.JObject;
            if (mapObj != null)
            {
                // Extract objects and save in LocalMapStore format
                var objects = mapObj["objects"];
                if (objects != null)
                {
                    // Build LocalMapStore-compatible JSON
                    string savedJson = "{\"mapName\":\"" + mapId + "\",\"savedAt\":0,\"objectCount\":" + (objects as Newtonsoft.Json.Linq.JArray)?.Count + ",\"objects\":" + objects.ToString() + "}";
                    string dir = Application.persistentDataPath + "/maps/";
                    if (!System.IO.Directory.Exists(dir)) System.IO.Directory.CreateDirectory(dir);
                    System.IO.File.WriteAllText(dir + mapId + ".json", savedJson);
                }
            }
            PlayerPrefs.SetString("SelectedMap", mapId);
            SceneManager.LoadScene("MultiGame");
        }
    }

    void DeleteMap(string mapName)
    {
        Debug.Log("[MapSelect] Delete: " + mapName);
        LocalMapStore.Delete(mapName);
        LoadMapList();
    }

    void OnDestroy()
    {
        if (_net != null) _net.OnServerMessage -= OnServerMsg;
    }
}
