using System;
using System.Collections;
using UnityEngine;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

public class NetworkManager : MonoBehaviour
{
    public static NetworkManager Instance { get; private set; }
    private WSClient _ws;
    private UDPClient _udp;
    private uint _seq;
    private Coroutine _heartbeatCoroutine;

    public bool IsConnected => _ws != null && _ws.IsConnected;
    public string connectionError { get; private set; }
    public UDPClient UdpClient => _udp;

    public event Action OnConnected;
    public event Action<string> OnDisconnected;
    public event Action<string, JObject> OnServerMessage;
    public event Action<ErrorPayload> OnErrorReceived;

    // Deferred to main thread
    private bool _pendingConnect, _pendingDisconnect;
    private string _dcReason, _udpIP;
    private int _udpPort;
    private System.Collections.Generic.Queue<(string type, JObject payload)> _msgQueue = new();

    void Awake()
    {
        var existing = FindObjectOfType<NetworkManager>();
        if (existing != null && existing != this) { Destroy(gameObject); return; }
        DontDestroyOnLoad(gameObject);
        Instance = this;
        _udp = GetComponent<UDPClient>();
        if (_udp == null) _udp = gameObject.AddComponent<UDPClient>();
    }

    void Start()
    {
        _ws = GetComponent<WSClient>();
        if (_ws == null) _ws = gameObject.AddComponent<WSClient>();
        _ws.OnConnected += () => { _pendingConnect = true; };
        _ws.OnDisconnected += (r) => { _dcReason = r; _pendingDisconnect = true; };
        _ws.OnMessageReceived += (t, p) => { _msgQueue.Enqueue((t, p)); };
    }

    private string _msgType;
    private JObject _msgPayload;
    private bool _pendingMsg;

    void Update()
    {
        if (_pendingConnect) { _pendingConnect = false; OnConnected?.Invoke(); }
        if (_pendingDisconnect) { _pendingDisconnect = false; OnDisconnected?.Invoke(_dcReason); HandleDisconnected(_dcReason); }
        while (_msgQueue.Count > 0) { var (t, p) = _msgQueue.Dequeue(); HandleMessage(t, p); }
    }

    public void Connect()
    {
        var ctx = GameContext.Instance;
        if (string.IsNullOrEmpty(ctx.serverIP)) return;
        _ = _ws.Connect(ctx.WsUrl);
    }

    public void JoinRoom(string joinCode)
    {
        SendToServer(MessageTypes.C_PLAYER_JOIN, new PlayerJoinPayload
        {
            player_id = GameContext.Instance.PlayerID,
            join_code = joinCode, display_name = GameContext.Instance.displayName,
            device_type = "Pico4Ultra", udp_port = 0
        });
    }

    public void GrabFlag(string flagId, Vector3 pos)
    {
        SendToServer(MessageTypes.C_GRAB_FLAG, new GrabFlagPayload
        {
            player_id = GameContext.Instance.PlayerID, flag_id = flagId,
            pos = new Vector3Json { x = pos.x, y = pos.y, z = pos.z }
        });
    }

    public void SendToServer(string type, object payload) { _ws.SendMessage(type, payload, _seq++); }

    private void HandleDisconnected(string reason)
    {
        _udp.Stop();
        if (_heartbeatCoroutine != null) { StopCoroutine(_heartbeatCoroutine); _heartbeatCoroutine = null; }
    }

    private IEnumerator HeartbeatLoop()
    {
        var wait = new WaitForSeconds(15f);
        while (_ws != null && _ws.IsConnected) { yield return wait; SendToServer(MessageTypes.C_HEARTBEAT, null); }
    }

    private void HandleMessage(string type, JObject payload)
    {
        switch (type)
        {
            case MessageTypes.S_PLAYER_JOINED:
                var j = payload.ToObject<PlayerJoinedPayload>();
                GameContext.Instance.SetRoom(GameContext.Instance.RoomID, j?.player_id);
                GameContext.Instance.Slot = j?.slot ?? -1;
                break;
            case MessageTypes.S_ROOM_CREATED:
                var rc = payload.ToObject<RoomCreatedPayload>();
                GameContext.Instance.RoomID = rc.room_id;
                break;
            case MessageTypes.S_ERROR:
                OnErrorReceived?.Invoke(payload.ToObject<ErrorPayload>());
                break;
        }
        OnServerMessage?.Invoke(type, payload);
    }

    // Called from OnConnected on main thread
    public void StartUdp(string ip, int port) 
    { 
        _udpIP = ip; _udpPort = port; 
        StartCoroutine(ConnectUdpNextFrame()); 
    }

    System.Collections.IEnumerator ConnectUdpNextFrame()
    {
        yield return null;
        if (_udp != null)
        {
            Debug.Log("[Net] UDP connecting to " + _udpIP + ":" + _udpPort);
            _udp.Connect(_udpIP, _udpPort);
        }
    }
    public void StartHeartbeat()
    {
        if (_heartbeatCoroutine != null) StopCoroutine(_heartbeatCoroutine);
        _heartbeatCoroutine = StartCoroutine(HeartbeatLoop());
    }

    void OnDestroy() { _udp?.Stop(); if (_heartbeatCoroutine != null) StopCoroutine(_heartbeatCoroutine); }
}
