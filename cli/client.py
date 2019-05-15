import http.client
import subprocess
import os
sc_api = "localhost:8080"
manifest_id = "shatter"
token_cmd = "oauth2local token"
surfaceFile = ""


res = subprocess.run(["oauth2local", "token"], stdout=subprocess.PIPE)
h = {"Authorization": "Bearer " + str(res.stdout)}

conn = http.client.HTTPConnection(sc_api)
conn.request("POST", "/stitch/"+manifest_id, headers=h, body=open(surfaceFile))
r1 = conn.getresponse()
print(r1)
