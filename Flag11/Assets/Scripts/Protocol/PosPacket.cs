using System;

[Serializable]
public class PosPacket
{
    public string room_id;
    public string player_id;
    public uint seq;
    public long ts;
    public float x;
    public float y;
    public float z;
    public float rot_y;
    public float head_pitch;
    public byte flags;

    public const byte FLAG_MOVING = 1;
    public const byte FLAG_GRABBING = 2;

    public bool IsMoving => (flags & FLAG_MOVING) != 0;
    public bool IsGrabbing => (flags & FLAG_GRABBING) != 0;
}

[Serializable]
public class RelayPacket
{
    public string from_player_id;
    public uint seq;
    public long server_ts;
    public float x;
    public float y;
    public float z;
    public float rot_y;
    public float head_pitch;
    public byte flags;
}