using UnityEngine;

public class CanvasXRSetup : MonoBehaviour
{
    public float planeDistance = 2f;

    private Canvas _canvas;

    void Start()
    {
        _canvas = GetComponent<Canvas>();
        if (_canvas == null) _canvas = gameObject.AddComponent<Canvas>();

        bool isXR = Application.platform == RuntimePlatform.Android && !Application.isEditor;
        Debug.Log($"[CanvasXR] isXR={isXR}");

        if (isXR)
        {
            _canvas.renderMode = RenderMode.ScreenSpaceCamera;

            var xrOrigin = FindObjectOfType<Unity.XR.CoreUtils.XROrigin>();
            if (xrOrigin != null && xrOrigin.Camera != null)
            {
                _canvas.worldCamera = xrOrigin.Camera;
                _canvas.planeDistance = planeDistance;
                Debug.Log($"[CanvasXR] ScreenSpaceCamera cam={xrOrigin.Camera.name} dist={planeDistance}");
            }
            else
            {
                Debug.LogError("[CanvasXR] XROrigin.Camera is NULL! Falling back to Camera.main");
                _canvas.worldCamera = Camera.main;
                _canvas.planeDistance = planeDistance;
            }
        }
        else
        {
            _canvas.renderMode = RenderMode.ScreenSpaceOverlay;
        }
    }
}