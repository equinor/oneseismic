import gql
import requests

import urllib.parse

def urljoin(base, path):
    """
    Parameters
    ----------
    base : str or None
        Base URL, e.g. 'https://oneseismic.equinor.com'
    path : str
        Path to add, e.g. result/<pid/lstream'

    Thin helper to remove a sharp edge from the urllib urljoin. The Base url
    must have a trailing / to preserve the last element of the path:
    >>> urljoin('base.com/path/sub', 'pid/stream')
    'base.com/path/pid/stream'
    >>> urljoin('base.com/path/sub/', 'pid/stream')
    'base.com/path/sub/pid/stream'
    """
    if base is not None:
        base += '/'
    return urllib.parse.urljoin(base, path)

class process:
    """Basic process handle

    This class implements the basics for a oneseismic process, initialised from
    a promise. It is intended as a tool library developers for managing URL
    generation and encoding the conventions and paths in oneseismic, but
    without the pinning the choice (or timing) of transport.

    Any reasonable action or query you would do against a process, either
    running, completed, or defunct, should in some respect be available through
    here. Library developers should use this and avoid hard-coding assumptions
    about endpoint layout.

    This class is "actionless" in the sense that it does not actually *perform*
    any network operations, it just provides the parameters required to do so.

    Parameters
    ----------
    query : str
        The query, e.g. sliceByIndex
    promise: dict
        The promise, parsed from json to dict
    """
    def __init__(self, query, promise):
        self.query = query
        self.path  = promise['url']
        self.key   = promise['key']
        self.pid   = self.path.split('/')[-1]
        self.promise = promise

    def __repr__(self):
        pid = self.pid
        query = self.query
        name = type(self).__name__
        return f'{name}(pid = {pid}, query = {query})'

    def headers(self):
        """Required headers for process requests

        Returns
        -------
        headers : dict
            Dict of { '<header>': '<val>' } that should be used when making
            requests
        """
        return { 'Authorization': f'Bearer {self.key}' }

    def status(self, baseurl = None):
        """URL to check process status

        Parameters
        ----------
        baseurl : str or None
            Base url to the oneseismic server. If this is None, this returns
            the relative path

        Returns
        -------
        status_url : str
            URL used to check the process status

        Examples
        --------
        >>> proc.status('https://oneseismic.equinor.com')
        'https://oneseismic.equinor.com/<pid>/status'
        """
        return urljoin(baseurl, f'{self.path}/status')

    def stream(self, baseurl = None):
        """URL to get payload stream

        Parameters
        ----------
        baseurl : str or None
            Base url to the oneseismic server. If this is None, this returns
            the relative path

        Returns
        -------
        payload_url : str
            URL to retrieve the payload

        Notes
        -----
        Code should not assume much about the content of the path, as the
        endpoint can be changed without notice. Accessing through stream()
        should always give a good URL.

        Examples
        --------
        >>> stream = proc.stream('https://oneseismic.equinor.com')
        >>> stream
        'https://oneseismic.equinor.com/<pid>/stream'
        >>> payload = requests.get(stream, headers = proc.headers())
        """
        # Right now this is hard-coded to stream, but this could move to /, be
        # read from the promise, or even be read from / in true REST fashion.
        return urljoin(baseurl, f'{self.path}/stream')

def procs_from_promises(response):
    """Make process instances from GraphQL response

    This is a utility function to automate the transformation of any GraphQL
    response into hydrated <proc> objects. This is a low-level library
    developer function and not intended for end users.

    It exists because writing this function ad-hoc is annoying and error prone.
    Only Promises are replaced, otherwise it is the identity function, in order
    to preserve the structure of the response.

    Parameters
    ----------
    response : dict
        Parsed GraphQL response as returned from the server

    Returns
    -------
    r : dict
        Dict with all Promises mapped to process

    Examples
    --------
    >>> response
    {'cube': { 'sliceByIndex': { 'key': '<key>', 'url': 'result/<pid>'}}}
    >>> procs_from_promises(response)
    {'cube': { 'sliceByIndex': proc(path = '<url>', pid = '<pid>')}}
    >>> r_with_list
    {'cube': {
        'sliceByIndex': { 'key': '<key>', 'url': 'result/<pid>'},
        'linenumbers: [[0, 1], [1, 2], [2, 3]],
    }}
    >>> procs_from_promises(r_with_list)
    {'cube': {
        'sliceByIndex': proc(path = '<url>', pid = '<pid>')}
        'linenumbers: [[0, 1], [1, 2], [2, 3]],
    }
    """
    if isinstance(response, dict):
        r = {}
        for key, val in response.items():
            try:
                r[key] = process(key, val)
            except (KeyError, TypeError, AttributeError):
                r[key] = procs_from_promises(val)
        return r
    elif isinstance(response, (list, tuple)):
        return [procs_from_promises(v) for v in response]
    else:
        return response

def filter_procs(response):
    """Get all processes

    This function walks the response and extracts all process objects. This is
    very useful when only data is queried (e.g. no line numbers and other
    contextual information), and particularly when there is only one item in
    the response. This function assumes procs_from_promises (or similar) has
    converted promises to process objects.

    This function is not meant to cover every edge case - rather, it intends to
    automate the tedious extraction of processes from responses in the common
    case.

    Parameters
    ----------
    response : dict
        GraphQL response that has been transformed by procs_from_promises

    Yields
    ------
    process

    See also
    --------
    procs_from_promises

    Examples
    --------
    >>> r_with_list
    {'cube': {
        'sliceByIndex': { 'key': '<key>', 'url': 'result/<pid>'},
        'linenumbers: [[0, 1], [1, 2], [2, 3]],
    }}
    >>> procs = procs_from_promises(response)
    >>> next(filter_procs(procs))
    process(path = '<url>', pid = '<pid>')
    >>> filtered = list(filter_procs(procs))
    >>> for proc in filtered:
    ...     r = requests.get(proc.status(), headers = proc.headers())
    ...     print(f'{proc.pid}: {r.content}')
    <pid1>: finished
    <pid2>: finished
    <pid3>: pending
    """
    # Maybe the isinstance() stuff can be replaced by try/except over
    # interfaces, in particular for the process type check.
    if isinstance(response, process):
        yield response
    elif isinstance(response, dict):
        for v in response.values():
            yield from filter_procs(v)
    elif isinstance(response, (list, tuple)):
        for v in response:
            yield from filter_procs(v)
