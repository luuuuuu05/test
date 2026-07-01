using UnityEngine;
using Unity.XR.CoreUtils;

public class LocalPlayer : MonoBehaviour
{
    [Header("Avatar (assign your own)")]
    public GameObject avatarPrefab;

    private UDPClient _udp;
    private Transform _cameraTransform;
    private int _sendCount;
    private float _nextLogTime;

    void Start()
    {
        _udp = NetworkManager.Instance != null ? NetworkManager.Instance.UdpClient : null;
        Debug.Log("[LocalPlayer] UDP=" + (_udp != null) + " RoomID=" + (GameContext.Instance?.RoomID ?? "null") + " PlayerID=" + (GameContext.Instance?.PlayerID ?? "null"));

        var xrOrigin = FindObjectOfType<XROrigin>();
        if (xrOrigin != null && xrOrigin.Camera != null)
        {
            _cameraTransform = xrOrigin.Camera.transform;
            if (avatarPrefab != null)
            {
                var avatar = Instantiate(avatarPrefab, _cameraTransform);
                avatar.transform.localPosition = new Vector3(0, -0.8f, 0.2f);
            }
        }
        else if (Camera.main != null)
            _cameraTransform = Camera.main.transform;

        if (_cameraTransform == null) _cameraTransform = transform;
    }

    void Update()
    {
        if (_udp == null || GameContext.Instance?.PlayerID == null || _cameraTransform == null) return;

        var pos = _cameraTransform.position;
        var euler = _cameraTransform.eulerAngles;

        _udp.SendPosition(new PosPacket
        {
            room_id = GameContext.Instance.RoomID,
            player_id = GameContext.Instance.PlayerID,
            x = pos.x, y = pos.y, z = pos.z,
            rot_y = euler.y, head_pitch = euler.x, flags = 0
        });
    }
}
