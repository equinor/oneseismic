import atexit
import ujson as json
import logging
import msal
import numpy as np
import os
import requests
from xdg import XDG_CACHE_HOME
import msgpack
import time

class ClientError(RuntimeError):
    pass

def assemble_slice(msg):
    parts = msgpack.unpackb(msg)

    # assume that all tiles have the same shape (which holds, at least for
    # now), so look up the shape from the first tile
    shape0, shape1 = parts[0]['shape']
    result = np.zeros(shape0 * shape1)

    for part in parts:
        for tile in part['tiles']:
            layout = tile
            dst = layout['initial-skip']
            chunk_size = layout['chunk-size']
            src = 0
            for _ in range(layout['iterations']):
                result[dst : dst + chunk_size] = tile['v'][src : src + chunk_size]
                src += layout['substride']
                dst += layout['superstride']

    return result.reshape((shape0, shape1))

class cube:
    """ Cube handle

    Constructing a cube object does not trigger any http calls as all properties
    are fetched lazily.
    """
    def __init__(self, id, client):
        self.client = client
        self.id = id
        self._dim0 = None
        self._dim1 = None
        self._dim2 = None

    def slice(self, dim, lineno):
        """ Fetch a slice

        Parameters
        ----------

        dim : int
            The dimension along which to slice
        lineno : int
            The line number we would like to fetch. This corresponds to the
            axis labels given in the dim<n> members. In order to fetch the nth
            surface allong the mth dimension use lineno = dim<m>[n].

        Returns
        -------

        slice : numpy.ndarray
        """
        resource = f"query/{self.id}/slice/{dim}/{lineno}"
        r = self.client.get(resource)
        if r.status_code != 200:
            if r.status_code == 404:
                if 'param.lineno' in r.text:
                    raise ClientError(r.text)
                if 'param.dimension' in r.text:
                    raise ClientError(r.text)
            raise ClientError(f'HTTP error {r.status_code}: {r.text}')

        header = r.json()
        status = header['result'] + '/status'

        auth = 'Bearer {}'.format(header['authorization'])
        extra_headers = { 'Authorization': auth }

        while True:
            r = self.client.get(status, extra_headers = extra_headers)
            response = r.json()

            # a poor man's progress bar
            print(response)

            if r.status_code == 200:
                result = response['location']
                r = self.client.get(result, extra_headers = extra_headers)
                if r.status_code == 200:
                    return assemble_slice(r.content)
                else:
                    raise RuntimeError(f'Error getting slice; {r.status_code} {r.text}')

            elif r.status_code == 202:
                status = response['location']
                # This sleep needs to go - polling should be optional and
                # controllable by the caller.
                time.sleep(1)

            else:
                raise RuntimeError(f'Unknown error; {r.status_code} {r.text}')

class azure_auth:
    def __init__(self, cache_dir=None):
        self.app = None
        self.scopes = None
        self.cache_dir = cache_dir

    def token(self):
        """ Loads a token from cache

        Loads a token that has previously been cached by login() or the
        oneseismic-login command.

        This function is designed to be executed non-interactively and will fail
        if the token can not be loaded from cache and refreshed without user
        interaction.
        """
        if not self.app:
            config_path = os.path.join(
                self.cache_dir or XDG_CACHE_HOME,
                "oneseismic",
                "config.json"
            )
            try:
                config = json.load(open(config_path))
            except FileNotFoundError:
                raise RuntimeError(
                    "No credentials found in cache. Log in "
                    "using oneseismic-login or login()"
                )

            cache_file = os.path.join(
                self.cache_dir or XDG_CACHE_HOME,
                "oneseismic",
                "accessToken.json"
            )
            cache = msal.SerializableTokenCache()

            cache.deserialize(open(cache_file, "r").read())
            atexit.register(
                lambda: open(cache_file, "w").write(cache.serialize())
            )

            self.app = msal.PublicClientApplication(
                config['client_id'],
                authority=config['auth_server'],
                token_cache=cache,
            )

            self.scopes = config['scopes']

        account = self.app.get_accounts()[0]
        result = self.app.acquire_token_silent(
            self.scopes,
            account=account
        )

        if "access_token" not in result:
            raise RuntimeError(
                "A token was found in cache, but it does not appear to "
                "be valid. Try logging in again using oneseismic-login "
                "or login()"
            )

        return {"Authorization": "Bearer " + result["access_token"]}


class client:
    def __init__(self, endpoint, auth=None, cache_dir=None):
        self.endpoint = endpoint
        self.auth = auth or azure_auth(cache_dir)

    def token(self):
        return self.auth.token()

    def get(self, resource, extra_headers = None):
        url = f"{self.endpoint}/{resource}"
        headers = self.token()
        if extra_headers is not None:
            headers.update(extra_headers)
        return requests.get(url, headers = headers)

    def list_cubes(self):
        """ Return a list of cube ids

        Returns
        -------

        cube_ids : list of strings
        """
        return self.get('')

    def cube(self, id):
        """ Get a cube handle

        Parameters
        ----------

        id : str
            The guid of the cube.

        Returns
        -------

        c : cube
        """
        return cube(id, self)
