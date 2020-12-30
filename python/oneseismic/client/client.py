import collections
import numpy as np
import requests
import msgpack
import time

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
    def __init__(self, guid, session):
        self.session = session
        self.guid = guid
        self._shape = None
        self._ijk = None

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

        self._shape = tuple(len(dim) for dim in self.ijk)
        return self._shape

    @property
    def ijk(self):
        """
        Notes
        -----
        The ijk is immutable and the result may be cached.

        The ijk name is temporary and will change without notice
        """
        if self._ijk is not None:
            return self._ijk

        resource = f'query/{self.guid}'
        r = self.session.get(resource)
        self._ijk = [
            [x for x in dim['keys']] for dim in r.json()['dimensions']
        ]
        return self._ijk

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
        resource = f"query/{self.guid}/slice/{dim}/{lineno}"
        proc = schedule(
            session = self.session,
            resource = resource,
        )

        return assemble_slice(proc.raw_result())

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
    def __init__(self, session, status_url, result_url):
        self.session = session
        self.status_url = status_url
        self.result_url = result_url
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

def schedule(session, resource):
    """Start a server-side process.

    This function centralises setting up a HTTP session and building the
    process object, whereas end-users should use methods on the outermost cube
    class.

    Parameters
    ----------
    session : requests.Session
        Session object with a get() for making http requests
    resource : str
        Resource to schedule, e.g. 'query/<id>/slice'

    Returns
    -------
    proc : process
        Process handle for monitoring status and getting the result

    Notes
    -----
    Scheduling a process manually is reserved for the implementation.
    """
    r = session.get(resource)

    body = r.json()
    auth = 'Bearer {}'.format(body['authorization'])
    s = http_session(session.base_url)
    s.headers.update({'Authorization': auth})

    return process(
        session = s,
        status_url = body['status'],
        result_url = body['location'],
    )

class http_session(requests.Session):
    """
    http_session provides some automation on top of the requests.Session type,
    to simplify http requests in more seismic-specific interfaces and logic.
    Methods also raise non-200 http status codes as exceptions.

    The http_session methods do not take absolute URLs, but relative URLs e.g.
    req.get(url = 'result/<pid>/status').

    Parameters
    ----------
    base_url : str
        The base url, schema + host, for the oneseismic service
    auth :
        Object to request up-to-date authorization headers from

    Notes
    -----
    This class is meant for internal use, to provide a clean boundary for
    low-level network-oriented code.
    """
    def __init__(self, base_url, tokens = None, *args, **kwargs):
        self.base_url = base_url
        self.tokens = tokens
        super().__init__(*args, **kwargs)

    def merge_auth_headers(self, kwargs):
        if self.tokens is None:
            return kwargs

        headers = self.tokens.headers()
        if 'headers' in kwargs:
            # unpack-and-set rather than just assigning the dictionary, in case
            # headers() starts returning more than just the Authorization
            # headers. This puts the power of definition where it belongs, and
            # keeps http_session oblivious to oneseismic specific header
            # expectations.
            #
            # If users at call-time explicitly set any of these headers,
            # respect them
            for k, v in headers.items():
                kwargs['headers'].setdefault(k, v)
        else:
            kwargs['headers'] = headers

        return kwargs

    def get(self, url, *args, **kwargs):
        """HTTP GET

        requests.Session.get, but raises exception for non-200 HTTP status
        codes. Authorization headers will be added to the request if
        http_session.tokens is available.

        This function will respect call-level custom headers, and only use
        http_session.tokens.headers() if not specified, similar to the requests
        API [1]_.

        Parameters
        ----------
        url : str
            Relative url to the resource, e.g. 'result/<pid>/status'

        Returns
        -------
        r : request.Response

        See also
        --------
        requests.get

        References
        ----------
        .. [1] https://requests.readthedocs.io/en/master/user/advanced/#session-objects

        Examples
        --------
        Defaulted and custom authorization:
        >>> session = http_session(url, tokens = tokens)
        >>> session.get('/needs-auth')
        >>> session.get('/needs-auth', headers = { 'Accept': 'text/html' })
        >>> session.get('/no-auth', headers = { 'Authorization': None })
        """
        kwargs = self.merge_auth_headers(kwargs)
        r = super().get(f'{self.base_url}/{url}', *args, **kwargs)
        r.raise_for_status()
        return r

    @staticmethod
    def fromconfig(cache_dir = None):
        """Create a new session from on-disk config

        Create a new http_sesssion with parameters and auth read from disk.
        This is a convenient constructor for most programs and uses outside of
        testing.

        Parameters
        ----------
        cache_dir : path or str, optional
            Configuration cache directory

        Returns
        -------
        session : http_session
            A ready-to-use http_session with authorization headers set
        """
        from ..login.login import config, tokens
        cfg = config(cache_dir = cache_dir).load()
        auth = tokens(cache_dir = cache_dir).load(cfg)
        return http_session(base_url = cfg['url'], tokens = auth)

def ls(session):
    """List available cubes

    List the cubes stored in oneseismic. The ids returned should all be valid
    arguments for the oneseismic.client.cube class.

    Parameters
    ----------
    session : oneseismic.http_session
        Session with authorization headers set

    Returns
    -------
    guids : iterable of str
        Cube GUIDs

    See also
    --------
    oneseismic.client.cube
    """
    return session.get('query').json()['links'].keys()

class cubes(collections.abc.Mapping):
    """Dict-like interface to cubes in the oneseismic subscription

    Parameters
    ----------
    session : http_session
    """
    def __init__(self, session):
        self.session = session
        self.cache = None

    def __getitem__(self, guid):
        if guid not in self.guids:
            raise KeyError(guid)
        return cube(guid, self.session)

    def __iter__(self):
        yield from self.guids

    def __len__(self):
        return len(self.guids)

    def sync(self):
        """Synchronize the set of guids in the subscription.

        It is generally only necessary to call this function once, but it can
        be called manually to get new IDs that have been added to the
        subscription since the client was created. For programs, it is
        Generally a better idea to create a new client.

        This is intended for internal use.
        """
        self.cache = ls(self.session)

    @property
    def guids(self):
        """Guids of cubes in subscription

        This is for internal use.

        All other functions should use this property to interact with guids, as
        it manages the cache.
        """
        if self.cache is None:
            self.sync()
        return self.cache

class cli:
    """User friendly access to oneseismic

    Access oneseismic services in a user-friendly manner with the cli class,
    suitable for programs, REPLs, and notebooks.

    Parameters
    ----------
    session : http_session
    """
    def __init__(self, session):
        self.session = session

    @property
    def cubes(self):
        return cubes(self.session)
