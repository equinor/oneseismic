# seismic_cloud_sdk.DefaultApi

All URIs are relative to *https://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**upload_surface**](DefaultApi.md#upload_surface) | **POST** /surface/{surfaceID} | 


# **upload_surface**
> str upload_surface(surface_id)



post surface file

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
api_instance = seismic_cloud_sdk.DefaultApi(seismic_cloud_sdk.ApiClient(configuration))
surface_id = 'surface_id_example' # str | File ID

try:
    api_response = api_instance.upload_surface(surface_id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->upload_surface: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **surface_id** | **str**| File ID | 

### Return type

**str**

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: application/octet-stream
 - **Accept**: application/octet-stream

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

