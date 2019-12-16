# seismic_cloud_sdk.ManifestApi

All URIs are relative to *https://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**download_manifest**](ManifestApi.md#download_manifest) | **GET** /manifest/{manifest_id} | 


# **download_manifest**
> StoreManifest download_manifest(manifest_id)



get manifest file

### Example
```python
from __future__ import print_function
import time
import seismic_cloud_sdk
from seismic_cloud_sdk.rest import ApiException
from pprint import pprint

# Configure API key authorization: ApiKeyAuth
configuration = seismic_cloud_sdk.Configuration()
configuration.api_key['Authorization'] = 'YOUR_API_KEY'
# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['Authorization'] = 'Bearer'

# create an instance of the API class
api_instance = seismic_cloud_sdk.ManifestApi(seismic_cloud_sdk.ApiClient(configuration))
manifest_id = 'manifest_id_example' # str | File ID

try:
    api_response = api_instance.download_manifest(manifest_id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling ManifestApi->download_manifest: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **manifest_id** | **str**| File ID | 

### Return type

[**StoreManifest**](StoreManifest.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/octet-stream

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

