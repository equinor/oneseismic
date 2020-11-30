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
        self._shape = None

    @property
    def shape(self):
        """ Shape of the cube

        N-element int-tuple.

        Notes
        -----
        The shape is immutable and the result may be cached.
        """
        if self._shape is not None:
            return self._shape

        resource = f"query/{self.id}"
        r = self.client.get(resource)
        self._shape = tuple(int(dim['size']) for dim in r.json()['dimensions'])
        return self._shape

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
        try:
            r = self.client.get(resource)
        except requests.HTTPError as e:
            response = e.response
            if response.status_code != requests.codes.not_found:
                raise
            # TODO: invalid argument?
            if 'param.lineno' in response.text:
                raise ClientError(response.text)
            if 'param.dimension' in response.text:
                raise ClientError(response.text)
            raise

        header = r.json()
        status = header['result'] + '/status'

        auth = 'Bearer {}'.format(header['authorization'])
        headers = { 'Authorization': auth }

        while True:
            r = self.client.get(status, headers = headers)
            response = r.json()

            # a poor man's progress bar
            print(response)

            if r.status_code == 200:
                result = response['location']
                r = self.client.get(result, headers = headers)
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
        if auth is None:
            auth = azure_auth(cache_dir)
        self.session = requests.Session()
        self.session.headers.update(auth.token())

    def get(self, resource, headers = None):
        url = f"{self.endpoint}/{resource}"
        r = self.session.get(url, headers = headers)
        r.raise_for_status()
        return r

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
