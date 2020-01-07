# seismic_cloud_sdk.StitchApi

All URIs are relative to *https://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**stitch**](StitchApi.md#stitch) | **GET** /stitch/{manifest_id}/{surface_id} | 
[**stitch_dim**](StitchApi.md#stitch_dim) | **GET** /stitch/{manifest_id}/dim/{dim}/{lineno} | 


# **stitch**
> str stitch(manifest_id, surface_id)



post surface query to stitch

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
api_instance = seismic_cloud_sdk.StitchApi(seismic_cloud_sdk.ApiClient(configuration))
manifest_id = 'manifest_id_example' # str | The id of a manifest
surface_id = 'surface_id_example' # str | The id of a surface

try:
    api_response = api_instance.stitch(manifest_id, surface_id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling StitchApi->stitch: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **manifest_id** | **str**| The id of a manifest | 
 **surface_id** | **str**| The id of a surface | 

### Return type

**str**

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/octet-stream

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **stitch_dim**
> ControllerBytes stitch_dim(manifest_id, dim, lineno)



post surface query to stitch

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
api_instance = seismic_cloud_sdk.StitchApi(seismic_cloud_sdk.ApiClient(configuration))
manifest_id = 'manifest_id_example' # str | The id of a manifest
dim = 56 # int | The dimension, either of 0,1,2
lineno = 56 # int | The line number

try:
    api_response = api_instance.stitch_dim(manifest_id, dim, lineno)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling StitchApi->stitch_dim: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **manifest_id** | **str**| The id of a manifest | 
 **dim** | **int**| The dimension, either of 0,1,2 | 
 **lineno** | **int**| The line number | 

### Return type

[**ControllerBytes**](ControllerBytes.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/octet-stream

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

