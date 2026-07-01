using System;
using System.Collections;
using System.Net;
using System.Net.Sockets;
using UnityEngine;

public class UDPClient : MonoBehaviour
{
    private UdpClient _udp;
    private uint _seq;
    private bool _isRunning;

    public event Action<RelayPacket> OnRelayReceived;

    public void Connect(string serverIP, int serverPort)
    {
        try
        {
            _udp = new UdpClient();
            _udp.Connect(new IPEndPoint(IPAddress.Parse(serverIP), serverPort));
            _isRunning = true;
            Debug.Log($"[UDP] Connected {serverIP}:{serverPort} (MessagePack)");
            StartCoroutine(ReceiveLoop());
        }
        catch (Exception ex)
        {
            Debug.LogError($"[UDP] Connect: {ex.Message}");
        }
    }

    public void SendPosition(PosPacket packet)
    {
        if (_udp == null || !_isRunning) return;
        try
        {
            packet.seq = _seq++;
            packet.ts = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();
            byte[] data = MsgPack.EncodePosPacket(packet);
            _udp.Send(data, data.Length);
        }
        catch (Exception ex)
        {
            Debug.LogError($"[UDP] Send: {ex.Message}");
        }
    }

    private IEnumerator ReceiveLoop()
    {
        _udp.Client.ReceiveTimeout = 1;
        while (_isRunning)
        {
            try
            {
                if (_udp != null && _udp.Client != null && _udp.Client.Poll(1, System.Net.Sockets.SelectMode.SelectRead))
                {
                    System.Net.IPEndPoint ep = null;
                    byte[] data = _udp.Receive(ref ep);
                    Debug.Log("[UDP] Got " + data.Length + " bytes from " + (ep?.ToString()));
                    var relay = MsgPack.DecodeRelayPacket(data);
                    if (relay != null)
                    {
                        Debug.Log("[UDP] Relay OK: " + relay.from_player_id + " pos=" + relay.x + "," + relay.y + "," + relay.z);
                        OnRelayReceived?.Invoke(relay);
                    }
                    else
                    {
                        var hex = System.BitConverter.ToString(data).Replace("-", " ");
                        Debug.Log("[UDP] Decode FAILED, len=" + data.Length + " hex=" + hex);
                    }
                }
            }
            catch (System.Exception ex)
            {
                Debug.LogWarning("[UDP] Recv: " + ex.Message);
            }
            yield return null;
        }
    }

    public void Stop()
    {
        _isRunning = false;
        _udp?.Close();
        _udp = null;
    }

    void OnDestroy() { Stop(); }
}