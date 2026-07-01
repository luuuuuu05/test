using UnityEngine;

public class EditorFallback : MonoBehaviour
{
    void Awake()
    {
#if UNITY_EDITOR
        Debug.Log("[EditorFallback] Editor mode - disabling XR");

        var pxrTypes = new[] {
            "PXR_Manager", "PXR_Hand", "PXR_Input", "PXR_Overlay",
            "PXR_SpatialAnchor", "PXR_MixedReality", "PXR_VideoSeeThrough",
            "PXR_PassThrough", "PXR_Boundary"
        };
        foreach (var c in GetComponents<MonoBehaviour>())
        {
            if (c == null || c == this) continue;
            var typeName = c.GetType().Name;
            foreach (var pxr in pxrTypes)
                if (typeName.StartsWith(pxr)) { c.enabled = false; break; }
        }

        var xrOrigin = GetComponent<Unity.XR.CoreUtils.XROrigin>();
        if (xrOrigin != null) xrOrigin.enabled = false;

        var inputActionMgr = GetComponent<UnityEngine.XR.Interaction.Toolkit.Inputs.InputActionManager>();
        if (inputActionMgr != null) inputActionMgr.enabled = false;

        var xrIM = FindObjectOfType<UnityEngine.XR.Interaction.Toolkit.XRInteractionManager>();
        if (xrIM != null) xrIM.enabled = false;

        var camObj = transform.Find("Camera Offset/Main Camera");
        if (camObj != null)
        {
            var cam = camObj.GetComponent<Camera>();
            if (cam != null) { cam.clearFlags = CameraClearFlags.SolidColor; cam.backgroundColor = new Color(0.1f, 0.1f, 0.2f, 1); cam.enabled = true; }
        }
#endif
    }
}