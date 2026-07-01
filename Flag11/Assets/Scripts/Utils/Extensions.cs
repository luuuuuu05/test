using System.Collections.Generic;

public static class Extensions
{
    public static string GetOr(this Dictionary<string, string> d, string k, string def)
    {
        return d != null && d.TryGetValue(k, out string v) ? v : def;
    }
}
