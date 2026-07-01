using System;
using System.Collections.Concurrent;
using System.Net.WebSockets;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using UnityEngine;
using Newtonsoft.Json.Linq;

public class WSClient : MonoBehaviour
{
    private ClientWebSocket _ws;
    private CancellationTokenSource _cts;
    private ConcurrentQueue<string> _recvQueue = new();
    private ConcurrentQueue<string> _sendQueue = new();

    private string _lastUrl;
    private int _reconnectAttempts;
    private const int MAX_RECONNECT = 3;
    private const float RECONNECT_DELAY = 2f;

    public bool IsConnected => _ws?.State == WebSocketState.Open;

    public event Action<string, JObject> OnMessageReceived;
    public event Action OnConnected;
    public event Action<string> OnDisconnected;

    private const int MAX_RECV_PER_FRAME = 10;

    public async Task Connect(string url, string roomId = null)
    {
        Debug.Log("[WS] Connect called. url=" + url + " roomId=" + roomId);
        _lastUrl = url;
        _reconnectAttempts = 0;
        await DoConnect(url, roomId);
    }

    private async Task DoConnect(string url, string roomId = null)
    {
        _cts = new CancellationTokenSource();
        _ws = new ClientWebSocket();

        if (!string.IsNullOrEmpty(roomId))
            _ws.Options.SetRequestHeader("X-Room-ID", roomId);
        _ws.Options.SetRequestHeader("X-Client-Type", "player");

        try
        {
            await _ws.ConnectAsync(new Uri(url), _cts.Token);
            Debug.Log($"[WS] Connected to {url}");
            OnConnected?.Invoke();
            _ = Task.Run(() => ReceiveLoop());
            _ = Task.Run(() => SendLoop());
        }
        catch (Exception ex)
        {
            Debug.LogError($"[WS] Connect failed: {ex.Message}");
            TryReconnect();
        }
    }

    private async void TryReconnect()
    {
        if (_reconnectAttempts >= MAX_RECONNECT)
        {
            Debug.LogWarning($"[WS] Max reconnect attempts reached ({MAX_RECONNECT})");
            OnDisconnected?.Invoke("Max reconnect attempts reached");
            return;
        }

        _reconnectAttempts++;
        Debug.Log($"[WS] Reconnecting in {RECONNECT_DELAY}s (attempt {_reconnectAttempts}/{MAX_RECONNECT})...");
        await Task.Delay((int)(RECONNECT_DELAY * 1000));

        _ = DoConnect(_lastUrl);
    }

    public void SendMessage(string type, object payload, uint seq = 0)
    {
        var msg = new JObject
        {
            ["type"] = type,
            ["seq"] = seq,
            ["ts"] = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds(),
            ["room_id"] = GameContext.Instance?.RoomID ?? "",
            ["payload"] = payload != null ? JToken.FromObject(payload) : new JObject()
        };

        _sendQueue.Enqueue(msg.ToString(Newtonsoft.Json.Formatting.None));
    }

    private async Task ReceiveLoop()
    {
        var buffer = new byte[4096];
        var msgBuffer = new StringBuilder();

        try
        {
            while (_ws?.State == WebSocketState.Open)
            {
                var result = await _ws.ReceiveAsync(new ArraySegment<byte>(buffer), _cts.Token);
                if (result.MessageType == WebSocketMessageType.Close)
                {
                    Debug.Log("[WS] Server closed connection");
                    break;
                }
                msgBuffer.Append(Encoding.UTF8.GetString(buffer, 0, result.Count));
                if (result.EndOfMessage)
                {
                    _recvQueue.Enqueue(msgBuffer.ToString());
                    msgBuffer.Clear();
                }
            }
        }
        catch (OperationCanceledException) { }
        catch (Exception ex)
        {
            Debug.LogError($"[WS] Receive error: {ex.Message}");
        }

        Debug.Log("[WS] Receive loop ended");
        TryReconnect();
    }

    private async Task SendLoop()
    {
        try
        {
            while (_ws?.State == WebSocketState.Open)
            {
                if (_sendQueue.TryDequeue(out var msg))
                {
                    var bytes = Encoding.UTF8.GetBytes(msg);
                    await _ws.SendAsync(new ArraySegment<byte>(bytes), WebSocketMessageType.Text, true, _cts.Token);
                }
                else
                {
                    await Task.Delay(5);
                }
            }
        }
        catch (OperationCanceledException) { }
        catch (Exception ex)
        {
            Debug.LogError($"[WS] Send error: {ex.Message}");
        }
    }

    void Update()
    {
        int processed = 0;
        while (processed++ < MAX_RECV_PER_FRAME && _recvQueue.TryDequeue(out var raw))
        {
            try
            {
                var json = JObject.Parse(raw);
                var type = json["type"]?.ToString() ?? "";
                var payload = json["payload"] as JObject;
                OnMessageReceived?.Invoke(type, payload);
            }
            catch (Exception ex)
            {
                Debug.LogError($"[WS] Parse error: {ex.Message}");
            }
        }
    }

    public async Task Disconnect()
    {
        _cts?.Cancel();
        if (_ws?.State == WebSocketState.Open)
        {
            try
            {
                await _ws.CloseAsync(WebSocketCloseStatus.NormalClosure, "Client closing", CancellationToken.None);
            }
            catch { }
        }
        _ws?.Dispose();
        _ws = null;
    }

    void OnDestroy()
    {
        _ = Disconnect();
    }
}