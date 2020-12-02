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
        proc = schedule(
            client = self.client,
            resource = resource,
        )

        return assemble_slice(proc.raw_result())

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

class process:
    """

    Maps conceptually to an observer of a process server-side. Comes with
    methods for querying status, completedness, and the final result.

    Parameters
    ----------
    host : str
        Hostname.
    session : request.Session
        A requests.Session-like with a get() method. Authorization headers
        should be set.
    status_url : str
        Relative path to the status endpoint.
    result_url : str
        Relative path to the result endpoint.

    Notes
    -----
    Constructing a process manually is reserved for the implementation.

    See also
    --------
    schedule
    """
    def __init__(self, host, session, status_url, result_url):
        self.session = session
        self.status_url = f'{host}/{status_url}'
        self.result_url = f'{host}/{result_url}'
        self.done = False

    def status(self):
        """ Processs status

        Retuns
        ------
        status : str
            Returns one of { 'working', 'finished' }

        Notes
        -----
        This function simply returns what the server responds with, so code
        inspecting the status should always have a fall-through case, in case
        the server is updated and returns something new.
        """
        r = self.session.get(self.status_url)
        response = r.json()

        if r.status_code == 200:
            self.done = True
            return response['status']

        if r.status_code == 202:
            return response['status']

        raise AssertionError(f'Unhandled status code f{r.status_code}')

    def raw_result(self):
        """
        The response body for result. This method should rarely be called
        directly, but can be useful for debugging, inspecting, or custom
        parsing.

        The function will block until the result is ready.
        """
        # TODO: optionally do a blocking read server-side
        # TODO: async/await support
        while not self.done:
            _ = self.status()
            if not self.done:
                time.sleep(1)

        # TODO: cache?
        r = self.session.get(self.result_url)
        return r.content

    def result(self):
        return self.assemble(self.raw_result())

    def assemble(self, body):
        """
        Assemble the response body into a suitable object. To be implemented by
        derived classes.
        """
        raise NotImplementedError

def schedule(client, resource):
    """Start a server-side process.

    This function centralises setting up a HTTP session and building the
    process object, whereas end-users should use methods on the outermost cube
    class.

    Returns
    -------
    proc : process
        Process handle for monitoring status and getting the result

    Notes
    -----
    Scheduling a process manually is reserved for the implementation.
    """
    r = client.get(resource)

    body = r.json()
    auth = 'Bearer {}'.format(body['authorization'])
    session = requests.Session()
    session.headers.update({'Authorization': auth})

    return process(
        host = client.endpoint,
        session = session,
        status_url = body['status'],
        result_url = body['location'],
    )

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
