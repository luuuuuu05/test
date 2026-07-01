using UnityEngine;

public class GameContext : MonoBehaviour
{
    public static GameContext Instance { get; private set; }

    [Header("Server")]
    public string serverIP = "";
    public int wsPort = 8081;
    public int udpPort = 9090;
    public int httpPort = 8080;

    public string AnchorUuid { get; private set; }

    public string RoomID { get; set; }
    public string PlayerID { get; set; }
    public int Slot { get; set; } = -1;

    [Header("Player")]
    public string displayName = "Player";

    public string WsUrl => $"ws://{serverIP}:{wsPort}/ws";

    void Awake()
    {
        if (Instance == null)
        {
            Instance = this;
            DontDestroyOnLoad(gameObject);
        }
        else
        {
            Destroy(gameObject);
            return;
        }

        Debug.unityLogger.logEnabled = true;
        Debug.Log($"[GameContext] Awake. Platform={Application.platform} Editor={Application.isEditor}");
        Application.logMessageReceived += (condition, stackTrace, type) =>
        {
            if (condition.StartsWith("[") && condition.Contains("]"))
                return; // Our tagged logs go through normal Debug.Log
        };
    }

    public void SetRoom(string roomId, string playerId)
    {
        RoomID = roomId;
        PlayerID = playerId;
        Debug.Log($"[GameContext] SetRoom room={roomId} player={playerId}");
    }

    public void SetAnchorUuid(string uuid)
    {
        AnchorUuid = uuid;
        Debug.Log($"[GameContext] Anchor UUID: {uuid}");
    }
}