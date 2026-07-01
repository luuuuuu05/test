using System.Collections.Generic;
using UnityEngine;

public class RemotePlayerManager : MonoBehaviour
{
    private Dictionary<string, RemotePlayer> _remotePlayers = new();
    public GameObject remotePlayerPrefab;
    private UDPClient _udp;

    void Start()
    {
        _udp = NetworkManager.Instance != null ? NetworkManager.Instance.UdpClient : null;
        if (_udp != null) _udp.OnRelayReceived += OnRelayReceived;
        if (NetworkManager.Instance != null)
            NetworkManager.Instance.OnServerMessage += OnServerMessage;
    }

    void OnServerMessage(string type, Newtonsoft.Json.Linq.JObject payload)
    {
        if (type == MessageTypes.S_PLAYER_JOINED)
        {
            var data = payload.ToObject<PlayerJoinedPayload>();
            if (data.player_id == GameContext.Instance.PlayerID) return;
            SpawnRemotePlayer(data.player_id, data.display_name,
                new Vector3(data.spawn_pos.x, data.spawn_pos.y, data.spawn_pos.z));
        }
    }

    void OnRelayReceived(RelayPacket pkt)
    {
        // Auto-create remote player from relay with CORRECT player ID
        if (!_remotePlayers.ContainsKey(pkt.from_player_id))
        {
            SpawnRemotePlayer(pkt.from_player_id, "Opponent",
                new Vector3(pkt.x, pkt.y, pkt.z));
        }

        if (_remotePlayers.TryGetValue(pkt.from_player_id, out var rp))
            rp.UpdatePosition(pkt);
    }

    void SpawnRemotePlayer(string playerId, string displayName, Vector3 pos)
    {
        if (_remotePlayers.ContainsKey(playerId)) return;

        GameObject obj;
        if (remotePlayerPrefab != null)
            obj = Instantiate(remotePlayerPrefab);
        else
        {
            obj = GameObject.CreatePrimitive(PrimitiveType.Capsule);
            obj.transform.localScale = new Vector3(0.5f, 1f, 0.5f);
            obj.GetComponent<Renderer>().material.color = Color.red;
        }

        obj.transform.position = pos;
        obj.name = "Remote_" + playerId;

        var rp = obj.GetComponent<RemotePlayer>();
        if (rp == null) rp = obj.AddComponent<RemotePlayer>();
        rp.playerID = playerId;
        rp.SetDisplayName(displayName);

        _remotePlayers[playerId] = rp;
    }

    void OnDestroy() { if (_udp != null) _udp.OnRelayReceived -= OnRelayReceived; }
}
