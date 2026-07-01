using System;
using System.Collections;
using UnityEngine;
using UnityEngine.Networking;

public class RestClient : MonoBehaviour
{
    public string ServerIP => GameContext.Instance != null ? GameContext.Instance.serverIP : "";
    public int HttpPort => 8080;

    private string BaseUrl => $"http://{ServerIP}:{HttpPort}";

    public event Action<string> OnRoomCreated; // room_id
    public event Action<string> OnError;

    public void CreateRoom(string jsonBody)
    {
        StartCoroutine(Post("/api/rooms", jsonBody, (json) =>
        {
            try
            {
                var resp = Newtonsoft.Json.Linq.JObject.Parse(json);
                var roomId = resp["room_id"]?.ToString() ?? "";
                OnRoomCreated?.Invoke(roomId);
            }
            catch (Exception e)
            {
                Debug.LogError($"[Rest] Parse error: {e.Message}");
            }
        }));
    }

    public void StartGame(string roomId)
    {
        StartCoroutine(Post($"/api/rooms/{roomId}/start", "{}", null));
    }

    public void StopGame(string roomId)
    {
        StartCoroutine(Post($"/api/rooms/{roomId}/stop", "{}", null));
    }

    public void GetHealth()
    {
        StartCoroutine(Get("/health", (json) =>
        {
            Debug.Log($"[Rest] Health: {json}");
        }));
    }

    private IEnumerator Post(string path, string body, Action<string> callback)
    {
        string url = BaseUrl + path;
        using (var req = new UnityWebRequest(url, "POST"))
        {
            byte[] bodyRaw = System.Text.Encoding.UTF8.GetBytes(body);
            req.uploadHandler = new UploadHandlerRaw(bodyRaw);
            req.downloadHandler = new DownloadHandlerBuffer();
            req.SetRequestHeader("Content-Type", "application/json");

            yield return req.SendWebRequest();

            if (req.result == UnityWebRequest.Result.Success)
            {
                Debug.Log("[Rest] POST " + path + " -> " + req.downloadHandler.text);
                callback?.Invoke(req.downloadHandler.text);
            }
            else
                Debug.LogError($"[Rest] POST {path} failed: {req.error}");
        }
    }

    private IEnumerator Get(string path, Action<string> callback)
    {
        string url = BaseUrl + path;
        using (var req = UnityWebRequest.Get(url))
        {
            yield return req.SendWebRequest();

            if (req.result == UnityWebRequest.Result.Success)
            {
                callback?.Invoke(req.downloadHandler.text);
            }
            else
                Debug.LogError($"[Rest] GET {path} failed: {req.error}");
        }
    }
}