import atexit
import ujson as json
import logging
import msal
import numpy as np
import os
from pathlib import Path
import requests


def assemble_slice(parts):
    tiles = parts['tiles']
    shape0 = parts['slice_shape']['dim0']
    shape1 = parts['slice_shape']['dim1']

    slice = np.zeros(shape0*shape1)

    for tile in tiles:
        layout = tile['layout']
        dst = layout['initial_skip']
        chunk_size = layout['chunk_size']
        src = 0
        for _ in range(layout['iterations']):
            slice[dst : dst + chunk_size] = tile['v'][src : src + chunk_size]
            src += layout['substride']
            dst += layout['superstride']

    return slice.reshape((shape0, shape1))


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

    @property
    def dim0(self):
        if not self._dim0:
            resource = f"{self.id}/slice/0"
            self._dim0 = self.client.get(resource)

        return self._dim0

    @property
    def dim1(self):
        if not self._dim1:
            resource = f"{self.id}/slice/1"
            self._dim1 = self.client.get(resource)

        return self._dim1

    @property
    def dim2(self):
        if not self._dim2:
            resource = f"{self.id}/slice/2"
            self._dim2 = self.client.get(resource)

        return self._dim2

    def slice(self, dim, lineno):
        """ Fetch a slice

        Parameters
        ----------

        dim : int
            The dimension allong which to slice
        lineno : int
            The line number we would like to fetch. This corresponds to the
            axis labels given in the dim<n> members. In order to fetch the nth
            surface allong the mth dimension use lineno = dim<m>[n].

        Returns
        -------

        slice : numpy.ndarray
        """
        resource = f"{self.id}/slice/{dim}/{lineno}"
        parts = self.client.get(resource)

        return assemble_slice(parts)


class azure_auth:
    def __init__(self):
        self.app = None
        self.scopes = None

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
                Path.home(),
                ".oneseismic",
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
                Path.home(),
                ".oneseismic",
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
    def __init__(self, endpoint, auth=azure_auth()):
        self.endpoint = endpoint
        self.auth = auth

    def token(self):
        return self.auth.token()

    def get(self, resource):
        url = f"{self.endpoint}/{resource}"
        r = requests.get(url, headers=self.token())

        if not r.status_code == 200:
            raise RuntimeError(
                f"Request {url} failed with status code {r.status_code}"
            )

        return json.loads(r.content)

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
