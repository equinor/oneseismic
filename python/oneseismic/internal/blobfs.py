import azure
import io
import numpy as np

from azure.storage.blob import BlobServiceClient

class blobfs:
    """Azure blob file system

    A file system that implements a directory like structure on top of the blob
    store. The blobfs keeps azure upload specific details out of programs, and
    expose the blob store as a regular IOBase interface.

    Parameters
    ----------
    client : azure.store.blob.BlobServiceClient
        Initiated connection to the blob store
    """
    def __init__(self, client):
        self.cwd = None
        self.cli = client

    def mkdir(self, path):
        """Make directory

        Parameters
        ----------
        path : pathlib.Path or str
            Directory to make

        Notes
        -----
        A 'directory' in blobfs means container. This function does not fail if
        the container already exists.
        """
        try:
            self.cli.create_container(name = path)
        except azure.core.exceptions.ResourceExistsError:
            pass

    def cd(self, path):
        """Change directory

        Parameters
        ----------
        path : pathlib.Path or str
            Directory to change to

        Notes
        -----
        This does not change os.getcwd()
        """
        self.cwd = self.cli.get_container_client(path)

    def open(self, name, mode = 'rb'):
        """Open file and return a stream

        Open a stream-like object to a blob. Note that the blob store is not a
        true operating system like file system, and the behaviour of the
        streams will differ slightly from some common assumptions. Most
        notably:

        1. The file streams are read-only or write-only
        2. Read-only streams are seekable
        3. Writable streams are not seekable, and will be written atomically

        Parameters
        ----------
        name : pathlib.Path or str
            File name to open
        mode : { 'rb', 'wb' }
            Mode in which the file is opened. Identical to the builtin open()
        """
        if mode in ['r', 'rb']:
            return blobread(self.cwd, name)

        if mode in ['w', 'wb']:
            return blobwrite(self.cwd, name)

        raise ValueError('mode must be rb or wb')

    @staticmethod
    def from_connection_string(cs):
        """Init filesystem from connection string

        Initialize a virtual blob filesystem based on a connection string [1]_.

        Parameters
        ----------
        cs : str
            Connection string

        References
        ----------
        .. [1] https://docs.microsoft.com/en-us/azure/storage/common/storage-configure-connection-string
        """
        client = BlobServiceClient.from_connection_string(cs)
        return blobfs(client = client)

    @staticmethod
    def from_url(account_url, credential):
        """Init filesystem from account URL

        Initialize a virtual blob filesystem based on an account URL and a
        credential. The credential can be a connection string or a shared
        access signature [1]_.

        Parameters
        ----------
        account_url : str
            Account URL, eg. https://<account>.blob.core.windows.net
        credential : str
            Credential, Shared Access Signature or connection string

        References
        ----------
        .. [1] https://docs.microsoft.com/en-us/azure/storage/common/storage-sas-overview
        """
        client = BlobServiceClient(
            account_url = account_url,
            credential = credential,
        )
        return blobfs(client = client)

class blobwrite(io.RawIOBase):
    def __init__(self, client, name):
        self.client = client.get_blob_client(name)

    def write(self, b):
        self.client.upload_blob(bytes(b))

    def writable(self):
        return True

    def writelines(self):
        raise NotImplementedError

    def seekable(self):
        return False

    def fileno(self):
        msg = 'blob store stream does not use file descriptors'
        raise IOError(msg)

    @property
    def closed(self):
        return False

class blobread(io.RawIOBase):
    def __init__(self, client, name, cachesize = 1e6):
        self.client = client.get_blob_client(name)
        self.cachesize = int(cachesize)

        self.cache_begin = 0
        self.cache_end = 0
        self.cache = None

        self.size = self.client.get_blob_properties().size
        self.pos = 0

    def readable(self):
        return True

    def readline(self, limit=-1):
        raise NotImplementedError

    def readlines(self, hint=-1):
        raise NotImplementedError

    def fileno(self):
        msg = 'blob store stream does not use file descriptors'
        raise IOError(msg)

    @property
    def closed(self):
        return False

    def read(self, n=-1):
        if n == -1:
            return self.readall()

        if n <= 0:
            raise ValueError('expected n >= 1, was {}'.format(n))

        if self.pos >= self.size:
            return b""

        nbytes = min(self.size - self.pos, n)

        # TODO: caching needs some love
        # TODO: dedicated cache object?
        cache_hit = self.cache_begin <= self.pos and self.pos + nbytes < self.cache_end

        if not cache_hit:
            readsize = max(self.cachesize, nbytes)
            download_stream = self.client.download_blob(self.pos, readsize)
            self.cache = download_stream.readall()
            self.cache_begin = self.pos
            self.cache_end = self.pos + readsize

        s = self.pos - self.cache_begin
        e = s + nbytes
        chunk = self.cache[s:e]

        self.pos += len(chunk)
        return chunk

    def readall(self):
        if self.pos >= self.size:
            return b""

        download_stream = self.client.download_blob(self.pos)
        chunk = download_stream.readall()
        self.pos = self.size
        return chunk

    def readinto(self, b):
        # TODO optimize? Issue #253
        # TODO implement read() in terms of readinto()
        chunk = self.read(b.size * b.itemsize)
        if len(chunk) == 0:
            return 0
        np.copyto(b, np.frombuffer(chunk, b.dtype).reshape(b.shape))
        return len(chunk)

    def seek(self, offset, whence=io.SEEK_SET):
        if whence == io.SEEK_SET:
            self.pos = offset
        elif whence == io.SEEK_CUR:
            self.pos += offset
        elif whence == io.SEEK_END:
            self.pos = self.size + offset
        else:
            msg = "unknown whence, expected one of {}, {}, {}"
            msg = msg.format("io.SEEK_SET", "io.SEEK_CUR", "io.SEEK_END")
            raise ValueError(msg)

    def seekable(self):
        return True
