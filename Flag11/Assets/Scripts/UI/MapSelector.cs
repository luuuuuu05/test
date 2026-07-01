using System.Collections.Generic;
using UnityEngine;
using UnityEngine.UI;

public class MapSelector : MonoBehaviour
{
    [Header("UI References")]
    public GameObject mapPanel;
    public Transform mapListParent;
    public Button mapItemPrefab;
    public Button newMapButton;
    public Button backButton;

    public List<MapInfo> availableMaps = new();

    [System.Serializable]
    public class MapInfo { public string mapId; public string mapName; public bool isOfficial; }

    private DevTest _dev;

    void Start()
    {
        _dev = GetComponent<DevTest>();
        if (mapPanel != null) mapPanel.SetActive(false);
        if (newMapButton != null) newMapButton.onClick.AddListener(OnNewMap);
        if (backButton != null) backButton.onClick.AddListener(() => UnityEngine.SceneManagement.SceneManager.LoadScene("MainMenu"));
    }

    public void Show()
    {
        LoadMapList();
        RefreshUI();
        if (mapPanel != null) mapPanel.SetActive(true);
    }

    public void Hide() { if (mapPanel != null) mapPanel.SetActive(false); }

    private void LoadMapList()
    {
        availableMaps.Clear();
        availableMaps.Add(new MapInfo { mapId = "official_default", mapName = "Default Arena", isOfficial = true });
        availableMaps.Add(new MapInfo { mapId = "official_small",   mapName = "Small Room",    isOfficial = true });
        availableMaps.Add(new MapInfo { mapId = "official_large",   mapName = "Large Space",   isOfficial = true });
        foreach (string name in LocalMapStore.ListAll())
            availableMaps.Add(new MapInfo { mapId = "local_" + name, mapName = name, isOfficial = false });
    }

    private void RefreshUI()
    {
        if (mapListParent == null) return;
        foreach (Transform child in mapListParent) Destroy(child.gameObject);

        foreach (var map in availableMaps)
        {
            if (mapItemPrefab == null) continue;
            var item = Instantiate(mapItemPrefab, mapListParent);
            var txt = item.GetComponentInChildren<Text>();
            if (txt != null) txt.text = (map.isOfficial ? "[Official] " : "[Custom] ") + map.mapName;
            string n = map.mapName;
            item.onClick.AddListener(() => SelectMap(n));
        }
    }

    private void SelectMap(string name)
    {
        Debug.Log("[MapSel] Selected: " + name);
        Hide();
        if (_dev != null) _dev.StartGame(name);
    }

    private void OnNewMap()
    {
        Debug.Log("[MapSel] New Map - entering build mode");
        Hide();
        if (_dev != null) _dev.StartBuildMode();
    }
}