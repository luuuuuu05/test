using UnityEngine;
using UnityEngine.UI;
using UnityEngine.SceneManagement;

public class JoinPanel : MonoBehaviour
{
    [Header("IP Buttons")]
    public Button ipBtn1, ipBtn2, ipBtn3;
    public Text currentIPText, statusText;
    public Button hostButton, joinButton, soloButton;

    private string _serverIP = "10.19.89.160";
    private NetworkManager _net;

    void Start()
    {
        _net = NetworkManager.Instance;
        if (_net != null) _net.OnServerMessage += OnServerMessage;

        if (ipBtn1 != null) ipBtn1.onClick.AddListener(() => SetIP("10.19.89.160"));
        if (ipBtn2 != null) ipBtn2.onClick.AddListener(() => SetIP("192.168.1.1"));
        if (ipBtn3 != null) ipBtn3.onClick.AddListener(() => SetIP("127.0.0.1"));
        if (hostButton != null) hostButton.onClick.AddListener(OnHost);
        if (joinButton != null) joinButton.onClick.AddListener(OnJoin);
        if (soloButton != null) soloButton.onClick.AddListener(OnSolo);
        SetIP(_serverIP);
    }

    void SetIP(string ip) { _serverIP = ip; if (currentIPText != null) currentIPText.text = "Server: " + ip; }

    public void OnHost()
    {
        var ctx = GetCtx();
        ctx.serverIP = _serverIP;
        ctx.PlayerID = "H_" + System.Guid.NewGuid().ToString("N").Substring(0, 6);
        ctx.displayName = "Host";
        Show("Connecting...");
        _net.Connect();
    }

    public void OnJoin()
    {
        var ctx = GetCtx();
        ctx.serverIP = _serverIP;
        ctx.PlayerID = "J_" + System.Guid.NewGuid().ToString("N").Substring(0, 6);
        ctx.displayName = "Guest";
        Show("Connecting...");
        _net.Connect();
    }

    public void OnSolo()
    {
        var ctx = GetCtx();
        ctx.serverIP = "";
        ctx.PlayerID = "SOLO_" + Random.Range(1000, 9999);
        ctx.displayName = "Solo";
        SceneManager.LoadScene("MapSelect");
    }

    void OnServerMessage(string type, Newtonsoft.Json.Linq.JObject payload)
    {
        if (type == "s_room_created")
        {
            var rc = payload.ToObject<RoomCreatedPayload>();
            GameContext.Instance.RoomID = rc.room_id;
            _net.JoinRoom(rc.join_code);
            // Start UDP + heartbeat now that we have a room
            var ctx = GameContext.Instance;
            _net.StartUdp(ctx.serverIP, ctx.udpPort);
            _net.StartHeartbeat();
            Show("Room " + rc.room_id + " joined");
        }
        else if (type == "s_player_joined")
        {
            Show("Player " + payload.ToObject<PlayerJoinedPayload>().player_count + "/2 joined!");
            StartCoroutine(GoMapSelect());
        }
        else if (type == "s_error")
            Show("Error: " + payload.ToObject<ErrorPayload>().message);
    }

    System.Collections.IEnumerator GoMapSelect() { yield return new WaitForSeconds(0.5f); SceneManager.LoadScene("MapSelect"); }

    GameContext GetCtx()
    {
        var ctx = GameContext.Instance ?? FindObjectOfType<GameContext>();
        if (ctx == null) { var go = new GameObject("GameContext"); ctx = go.AddComponent<GameContext>(); DontDestroyOnLoad(go); }
        return ctx;
    }

    void Show(string msg) { if (statusText != null) statusText.text = msg; Debug.Log("[JoinPanel] " + msg); }
    void OnDestroy() { if (_net != null) _net.OnServerMessage -= OnServerMessage; }
}
