using System;
using System.IO;
using System.Text;

public static class MsgPack
{
    public static byte[] EncodePosPacket(PosPacket pkt)
    {
        using var ms = new MemoryStream();
        ms.WriteByte(0x9A); // fixarray 10
        WriteStr(ms, pkt.room_id);
        WriteStr(ms, pkt.player_id);
        WriteU32(ms, pkt.seq);
        WriteI64(ms, pkt.ts);
        WriteF32(ms, pkt.x);
        WriteF32(ms, pkt.y);
        WriteF32(ms, pkt.z);
        WriteF32(ms, pkt.rot_y);
        WriteF32(ms, pkt.head_pitch);
        WriteU8(ms, pkt.flags);
        return ms.ToArray();
    }

    public static RelayPacket DecodeRelayPacket(byte[] data)
    {
        try
        {
            int pos = 0;
            if (data.Length < 3) return null;
            
            // Skip array header (0x99 = fixarray 9)
            byte header = data[pos];
            if ((header & 0xF0) != 0x90) return null; // not fixarray
            pos++;

            var pkt = new RelayPacket
            {
                from_player_id = ReadStr(data, ref pos),
                seq = ReadAnyUInt(data, ref pos),
                server_ts = ReadAnyInt(data, ref pos),
                x = ReadAnyFloat(data, ref pos),
                y = ReadAnyFloat(data, ref pos),
                z = ReadAnyFloat(data, ref pos),
                rot_y = ReadAnyFloat(data, ref pos),
                head_pitch = ReadAnyFloat(data, ref pos),
                flags = ReadU8(data, ref pos)
            };
            return pkt;
        }
        catch { return null; }
    }

    // ---- Writers ----
    static void WriteStr(MemoryStream ms, string s)
    {
        byte[] b = Encoding.UTF8.GetBytes(s ?? "");
        int len = b.Length;
        if (len <= 31) ms.WriteByte((byte)(0xA0 | len));
        else if (len <= 255) { ms.WriteByte(0xD9); ms.WriteByte((byte)len); }
        else { ms.WriteByte(0xDA); ms.WriteByte((byte)(len >> 8)); ms.WriteByte((byte)len); }
        ms.Write(b, 0, len);
    }

    static void WriteU32(MemoryStream ms, uint v) { ms.WriteByte(0xCE); WriteBE(ms, v, 4); }
    static void WriteI64(MemoryStream ms, long v) { ms.WriteByte(0xD3); WriteBE(ms, (ulong)v, 8); }
    static void WriteF32(MemoryStream ms, float v)
    {
        ms.WriteByte(0xCA);
        byte[] b = BitConverter.GetBytes(v);
        if (BitConverter.IsLittleEndian) Array.Reverse(b);
        ms.Write(b, 0, 4);
    }
    static void WriteU8(MemoryStream ms, byte v) { ms.WriteByte(v); }
    static void WriteBE(MemoryStream ms, ulong v, int bytes)
    {
        for (int i = bytes - 1; i >= 0; i--) ms.WriteByte((byte)(v >> (8 * i)));
    }

    // ---- Flexible Readers ----
    static string ReadStr(byte[] data, ref int pos)
    {
        byte b = data[pos++];
        int len;
        if ((b & 0xE0) == 0xA0) len = b & 0x1F;
        else if (b == 0xD9) len = data[pos++];
        else if (b == 0xDA) { len = (data[pos] << 8) | data[pos + 1]; pos += 2; }
        else throw new Exception("Bad str tag: " + b.ToString("X2"));
        string s = Encoding.UTF8.GetString(data, pos, len);
        pos += len;
        return s;
    }

    static uint ReadAnyUInt(byte[] data, ref int pos)
    {
        byte b = data[pos++];
        if ((b & 0x80) == 0) return b;                   // positive fixint
        if (b == 0xCC) return data[pos++];                // uint8
        if (b == 0xCD) { uint v = (uint)((data[pos] << 8) | data[pos + 1]); pos += 2; return v; } // uint16
        if (b == 0xCE) { uint v = (uint)ReadBE(data, ref pos, 4); return v; } // uint32
        throw new Exception("Bad uint tag: " + b.ToString("X2"));
    }

    static long ReadAnyInt(byte[] data, ref int pos)
    {
        byte b = data[pos++];
        if ((b & 0x80) == 0) return b;                   // positive fixint
        if (b == 0xD3) return (long)ReadBE(data, ref pos, 8); // int64
        if (b == 0xCF) { return (long)ReadBE(data, ref pos, 8); } // uint64
        throw new Exception("Bad int tag: " + b.ToString("X2"));
    }

    static float ReadAnyFloat(byte[] data, ref int pos)
    {
        byte b = data[pos++];
        if (b == 0xCA) // float32
        {
            byte[] fb = new byte[4];
            fb[0] = data[pos++]; fb[1] = data[pos++]; fb[2] = data[pos++]; fb[3] = data[pos++];
            if (BitConverter.IsLittleEndian) Array.Reverse(fb);
            return BitConverter.ToSingle(fb, 0);
        }
        if (b == 0xCB) // float64
        {
            byte[] fb = new byte[8];
            for (int i = 0; i < 8; i++) fb[i] = data[pos++];
            if (BitConverter.IsLittleEndian) Array.Reverse(fb);
            return (float)BitConverter.ToDouble(fb, 0);
        }
        throw new Exception("Bad float tag: " + b.ToString("X2") + " at pos " + (pos - 1));
    }

    static byte ReadU8(byte[] data, ref int pos) => data[pos++];

    static ulong ReadBE(byte[] data, ref int pos, int bytes)
    {
        ulong v = 0;
        for (int i = 0; i < bytes; i++) v = (v << 8) | data[pos++];
        return v;
    }
}
