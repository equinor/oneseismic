import sys
import json
import logging

from dotenv import load_dotenv
import requests
import msal
import atexit
import os
from pathlib import Path

load_dotenv()

cache_file = os.path.join(Path.home(), ".oneseismic")
cache = msal.SerializableTokenCache()
if os.path.exists(cache_file):
    cache.deserialize(open(cache_file, "r").read())
atexit.register(lambda: open(cache_file, "w").write(cache.serialize()))
config = {
    "client_id": os.getenv("CLIENT_ID"),
    "authority": os.getenv("AUTHSERVER"),
    "scopes": ["https://storage.azure.com/user_impersonation"],
}

# Create a preferably long-lived app instance which maintains a token cache.
app = msal.PublicClientApplication(
    config["client_id"], authority=config["authority"], token_cache=cache,
)

# The pattern to acquire a token looks like this.
result = None

# Note: If your device-flow app does not have any interactive ability, you can
#   completely skip the following cache part. But here we demonstrate it anyway.
# We now check the cache to see if we have some end users signed in before.
accounts = app.get_accounts()
if accounts:
    logging.info("Account(s) exists in cache, probably with token too. Let's try.")
    # print("Pick the account you want to use to proceed:")
    # for a in accounts:
    #     print(a)
    # Assuming the end user chose this one
    chosen = accounts[0]
    # Now let's try to find a token in cache for this account
    result = app.acquire_token_silent(config["scopes"], account=chosen)

if not result:
    logging.info("No suitable token exists in cache. Let's get a new one from AAD.")

    flow = app.initiate_device_flow(config["scopes"])
    if "user_code" not in flow:
        raise ValueError(
            "Fail to create device flow. Err: %s" % json.dumps(flow, indent=4)
        )

    print(flow["message"])
    sys.stdout.flush()  # Some terminal needs this to ensure the message is shown

    # Ideally you should wait here, in order to save some unnecessary polling
    # input(
    #     "Press Enter after signing in from another device to proceed, CTRL+C to abort."
    # )

    result = app.acquire_token_by_device_flow(flow)  # By default it will block
    # You can follow this instruction to shorten the block time
    #    https://msal-python.readthedocs.io/en/latest/#msal.PublicClientApplication.acquire_token_by_device_flow
    # or you may even turn off the blocking behavior,
    # and then keep calling acquire_token_by_device_flow(flow) in your own customized loop.

if "access_token" in result:
    # Calling graph using the access token
    graph_data = requests.get(  # Use token to call downstream service
        sys.argv[1], headers={"Authorization": "Bearer " + result["access_token"]},
    )

    if graph_data.status_code == 200:
        print(graph_data.json())
    else:
        print(graph_data)

    # print("Graph API call result: %s" % json.dumps(graph_data, indent=2))
else:
    print(result.get("error"))
    print(result.get("error_description"))
    print(result.get("correlation_id"))  # You may need this when reporting a bug
