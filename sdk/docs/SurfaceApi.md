# seismic_cloud_sdk.SurfaceApi

All URIs are relative to *https://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**download_surface**](SurfaceApi.md#download_surface) | **GET** /surface/{surfaceID} | 
[**list_surfaces**](SurfaceApi.md#list_surfaces) | **GET** /surface/ | 


# **download_surface**
> str download_surface(surface_id)



get surface file

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
api_instance = seismic_cloud_sdk.SurfaceApi(seismic_cloud_sdk.ApiClient(configuration))
surface_id = 'surface_id_example' # str | File ID

try:
    api_response = api_instance.download_surface(surface_id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling SurfaceApi->download_surface: %s\n" % e)
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

 - **Content-Type**: Not defined
 - **Accept**: application/octet-stream

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_surfaces**
> list[StoreSurfaceMeta] list_surfaces()



get list of available surfaces

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
api_instance = seismic_cloud_sdk.SurfaceApi(seismic_cloud_sdk.ApiClient(configuration))

try:
    api_response = api_instance.list_surfaces()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling SurfaceApi->list_surfaces: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**list[StoreSurfaceMeta]**](StoreSurfaceMeta.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

