"""
This module provides functions to help programs parse arguments, such as
understanding and splitting connection strings and file paths.

Properly parsing arguments can be ugly and error prone, and some types of
arguments are re-used with identical semantics across programs.
"""

from pathlib import Path
from urllib.parse import urlparse
from urllib.parse import urlunparse

from .blobfs import blobfs
from .localfs import localfs

def add_auth_args(parser, direction):
    """
    """
    if direction not in ['input', 'output']:
        msg = 'Bad auth direction - this is a programmer error'
        raise ValueError(msg)

    group = parser.add_argument_group(
        f'{direction}-auth',
        description = f'''
        Auth for {direction} files. Auth can be encoded in the URL, or
        specified explicitly.
        '''
    )
    group.add_argument(
        f'--{direction}-auth-method',
        choices = ['url', 'connection-string', 'credentials'],
        default = None,
        help = '''
        Method of authentication. By default, it is assumed authentication is
        encoded in the URL.
        '''
    )
    group.add_argument(
        f'--{direction}-connection-string',
        type = str,
    )
    group.add_argument(
        f'--{direction}-credentials',
        type = str,
    )

def blobfs_from_args(url, method, connstr, creds):
    """Initialize a blobfs from parsed args

    Try to initialize a blobfs from parsed arguments. No files are opened, and
    this function is not sensitive the files being input or output files.
    Instead, it is simple automation on how the arguments are interpreted, with
    fallbacks and exclusive options.

    If a method is explicitly specified, that method willfor authentication. If
    a no method is specified, and either the connection-string or the
    credentials are specified, then that method will be used.  If neither
    method, connection string, nor credentials are specified, it is assumed it
    is encoded in the URL. See the docs [1]_ for the blob client.

    Parameters
    ----------
    url : str
        Storage account URL
    method : str
        Method to use for authentication
    connstr : str
        Connection string
    creds : str
        Credentials, either SAS [2]_ or account key

    Returns
    -------
    fs : blobfs

    Raises
    ------
    ValueError
        If the url is not an url

    Notes
    -----
    It is quite easy to pass an URL or an authentication which Azure will
    reject. In is quite infeasible to distinguish good and bad URLs without
    actually making the call, which means authentication errors won't happen
    until the blobfs tries to connect to Azure. The absence of an exception in
    this function does not mean all input parameters are good.

    References
    ----------
    .. [1] https://docs.microsoft.com/en-us/python/api/azure-storage-blob/azure.storage.blob.blobserviceclient?view=azure-python#constructor
    .. [2] https://docs.microsoft.com/en-us/azure/storage/common/storage-sas-overview

    Examples
    --------
    >>> args = parser.parse_args()
    >>> infs = blobfs_from_args(
        args.src,
        args.input_auth_method,
        args.input_connection_string,
        args.input_credentials,
    )
    >>> outfs = blobfs_from_args(
        args.dst,
        args.output_auth_method,
        args.output_connection_string,
        args.output_credentials,
    )
    """
    if method is None:
        if connstr is None and creds is None:
            # TODO: maybe warn of default path being taken?
            method = 'url'
        elif connstr is not None:
            method = 'connection-string'
        elif creds is not None:
            method = 'credentials'
        else:
            raise AssertionError('Unhandled case')

    if method in ['url', 'credentials']:
        parsed = urlparse(url)
        if parsed.scheme not in ['http', 'https']:
            raise ValueError('not a blob url')

        # If the credentials are encoded in the URL, either as a SAS token or a
        # URL-encoded account key, then creds should be None anyway, and the
        # case of url-encoded and explicit credentials end up being the same

        # Only keep the first part of the path, in case it specifies the
        # container and blob.
        unparsed = (
            parsed.scheme,
            parsed.netloc,
            '/'.join(parsed.path.split('/')[:2]),
            parsed.params,
            parsed.query,
            parsed.fragment
        )
        return blobfs.from_url(urlunparse(unparsed), credential = creds)
    elif method == 'connection-string':
        return blobfs.from_connection_string(connstr)
    else:
        raise AssertionError('Unhandled case')

def localfs_from_args(path):
    if path is None:
        path = Path()
    path = Path(path)
    if path.is_absolute():
        root = Path('/')
    else:
        root = Path('.')
    return localfs(root)

def get_blob_path(url):
    """
    Extract the blob path from an URL.

    Returns
    -------
    container : str
    blob : str

    Examples
    --------
    >>> get_blob_path('https://storage.net/crate/fname')
    ('crate', 'fname')
    >>> get_blob_path('https://storage.net/crate/fname?key=5aef2c')
    ('crate', 'fname')
    >>> get_blob_path('https://storage.net/crate/fname?key=5aef2c')
    ('crate', 'fname')
    """
    parsed = urlparse(url)
    path = parsed.path.split('/')
    container = path[1]
    blob = '/'.join(path[2:])
    return container, blob
