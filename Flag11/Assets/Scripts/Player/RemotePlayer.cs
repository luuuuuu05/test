using System.Collections.Generic;
using UnityEngine;

public class RemotePlayer : MonoBehaviour
{
    public string playerID;

    [Header("Avatar Model (assign your own)")]
    public GameObject avatarPrefab;

    private GameObject _avatar;
    private Queue<(long localTs, Vector3 pos)> _posBuffer = new();
    private Vector3 _targetPos;
    private const float INTERP_DELAY = 0.1f;

    void Start()
    {
        if (avatarPrefab != null)
        {
            _avatar = Instantiate(avatarPrefab, transform);
        }
        else
        {
            // Placeholder capsule
            _avatar = GameObject.CreatePrimitive(PrimitiveType.Capsule);
            _avatar.transform.SetParent(transform);
            _avatar.transform.localPosition = new Vector3(0, 0.5f, 0);
            _avatar.transform.localScale = new Vector3(0.3f, 0.5f, 0.3f);
            _avatar.name = "Avatar_Placeholder";
        }

        // Tag avatar so user can replace
        if (_avatar != null) _avatar.tag = "EditorOnly";
    }

    public void SetDisplayName(string name)
    {
        this.name = "Remote_" + name;
    }

    public void UpdatePosition(RelayPacket pkt)
    {
        _targetPos = new Vector3(pkt.x, pkt.y, pkt.z);
        _posBuffer.Enqueue((System.DateTimeOffset.UtcNow.ToUnixTimeMilliseconds(), _targetPos));

        while (_posBuffer.Count > 5)
            _posBuffer.Dequeue();

        var rot = _avatar != null ? _avatar.transform.rotation.eulerAngles : transform.rotation.eulerAngles;
        rot.y = pkt.rot_y;
        if (_avatar != null) _avatar.transform.rotation = Quaternion.Euler(rot);
    }

    void Update()
    {
        if (_posBuffer.Count == 0) return;

        var now = System.DateTimeOffset.UtcNow.ToUnixTimeMilliseconds();
        while (_posBuffer.Count > 1 && _posBuffer.Peek().localTs < now - (long)(INTERP_DELAY * 1000))
            _posBuffer.Dequeue();

        if (_posBuffer.Count > 0)
            transform.position = Vector3.Lerp(transform.position, _posBuffer.Peek().pos, Time.deltaTime * 10f);
    }
}