import os

from seismic_cloud_sdk import ApiClient, Configuration, ManifestApi

config = Configuration()
config.host = os.environ["HOST"]
config.api_key = {"Authorization": os.environ["API_TOKEN"]}
config.api_key_prefix = {"Authorization": "Bearer"}

config.debug = True

client = ApiClient(configuration=config)
manifest_api = ManifestApi(api_client=client)
