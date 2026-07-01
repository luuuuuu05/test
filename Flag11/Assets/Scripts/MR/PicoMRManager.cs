using System;
using System.Collections;
using UnityEngine;

public class PicoMRManager : MonoBehaviour
{
    [Header("Video See-Through")]
    public bool enableVST = true;

    [Header("Hand Tracking")]
    public bool enableHandTracking = true;

    [Header("Spatial Anchor")]
    public bool enableSharedAnchor = true;
    public float driftCorrectionInterval = 0.5f;

    private ulong _anchorHandle;
    private Guid _anchorUuid;
    private float _driftTimer;
    private bool _isAndroid;

    public string SharedAnchorUuid => _anchorUuid.ToString();

    void Awake()
    {
        _isAndroid = Application.platform == RuntimePlatform.Android;
        Debug.Log($"[PicoMR] Awake isAndroid={_isAndroid}");

        RenderSettings.skybox = null;

        var xrOrigin = GetComponent<Unity.XR.CoreUtils.XROrigin>();
        if (xrOrigin != null && xrOrigin.Camera != null)
        {
            xrOrigin.Camera.clearFlags = CameraClearFlags.SolidColor;
            xrOrigin.Camera.backgroundColor = new Color(0, 0, 0, 0);
        }
    }

    IEnumerator Start()
    {
        if (!_isAndroid)
        {
            Debug.Log("[PicoMR] Editor mode");
            yield break;
        }

        // Wait for XR session to be fully ready before enabling VST
        yield return new WaitForSeconds(0.5f);

        if (enableVST)
        {
            Unity.XR.PXR.PXR_Manager.EnableVideoSeeThrough = true;
            Debug.Log("[PicoMR] VST enabled (delayed)");
        }

        if (enableHandTracking)
        {
            Debug.Log("[PicoMR] Hand tracking not required (controller-only mode)");
        }

        if (enableSharedAnchor)
        {
            var result = Unity.XR.PXR.PXR_MixedReality.StartSenseDataProvider(
                Unity.XR.PXR.PxrSenseDataProviderType.SpatialAnchor);
            Debug.Log($"[PicoMR] SenseDataProvider: {result}");
            yield return SetupSharedAnchor();
        }
    }

    void OnApplicationPause(bool paused)
    {
        if (!paused && enableVST && _isAndroid)
        {
            StartCoroutine(ReEnableVST());
        }
    }

    private IEnumerator ReEnableVST()
    {
        yield return new WaitForSeconds(0.3f);
        Unity.XR.PXR.PXR_Manager.EnableVideoSeeThrough = true;
        Debug.Log("[PicoMR] VST re-enabled after pause");
    }

    private IEnumerator SetupSharedAnchor()
    {
        yield return new WaitForSeconds(0.3f);
        Debug.Log("[PicoMR] Creating spatial anchor...");

        var createTask = Unity.XR.PXR.PXR_MixedReality.CreateSpatialAnchorAsync(transform.position, transform.rotation);
        yield return new WaitUntil(() => createTask.IsCompleted);

        if (createTask.Result.result == Unity.XR.PXR.PxrResult.SUCCESS)
        {
            _anchorHandle = createTask.Result.anchorHandle;
            _anchorUuid = createTask.Result.uuid;
            Debug.Log($"[PicoMR] Anchor: {_anchorUuid}");

            var persistTask = Unity.XR.PXR.PXR_MixedReality.PersistSpatialAnchorAsync(_anchorHandle);
            yield return new WaitUntil(() => persistTask.IsCompleted);

            var uploadTask = Unity.XR.PXR.PXR_MixedReality.UploadSpatialAnchorAsync(_anchorHandle);
            yield return new WaitUntil(() => uploadTask.IsCompleted);

            if (uploadTask.Result.result == Unity.XR.PXR.PxrResult.SUCCESS)
            {
                _anchorUuid = uploadTask.Result.uuid;
                GameContext.Instance?.SetAnchorUuid(_anchorUuid.ToString());
            }
            Debug.Log($"[PicoMR] Anchor setup complete");
        }
    }

    void FixedUpdate()
    {
        if (_anchorHandle == 0 || !_isAndroid) return;
        _driftTimer += Time.fixedDeltaTime;
        if (_driftTimer >= driftCorrectionInterval)
        {
            _driftTimer = 0f;
            var locateResult = Unity.XR.PXR.PXR_MixedReality.LocateAnchor(_anchorHandle, out Vector3 pos, out Quaternion rot);
            if (locateResult == Unity.XR.PXR.PxrResult.SUCCESS)
            {
                transform.position = pos; transform.rotation = rot;
            }
        }
    }

    void OnDestroy()
    {
        if (_anchorHandle != 0 && _isAndroid)
            Unity.XR.PXR.PXR_MixedReality.DestroyAnchor(_anchorHandle);
    }
}