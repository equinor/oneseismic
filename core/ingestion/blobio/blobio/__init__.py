import io
import numpy as np

class BlobIO:
    """
    An io.RawIO like file object for azure blob store that provides a familiar
    interface for python programs
    """

    def __init__(self, blob_service, container, cachesize=1e6):
        """
        Parameters
        ----------
        blob_service : azure.storage.blob.*
            An azure.storage.blob BlobServiceClient object
        container : str_like
            Blob container name
        Example
        --------
        >>> from azure.storage.blob import BlobServiceClient
        >>> blob_service = BlobServiceClient.from_connection_string(connect_str)
        >>> blobio = BlobIO(blob_service=blob_service, container="dir")
        >>> stream = blobio.open(filename="file.csv")
        """
        self.blob_service = blob_service
        self.container = str(container)
        self.cachesize = cachesize

        self.cache_begin = 0
        self.cache_end = 0
        self.cache = None

    def open(self, blobname):
        """
        Parameters
        ----------
        filename : str_like
            Blob file name
        """
        self.pos = 0
        self.blobname = str(blobname)
        container_client = self.blob_service.get_container_client(self.container)
        self.blob_client = container_client.get_blob_client(self.blobname)
        self.size = self.blob_client.get_blob_properties().size
        return self


    @property
    def closed(self):
        return False

    def fileno(self):
        msg = "blob store stream does not use file descriptors"
        raise IOError(msg)

    def isatty(self):
        return False

    def readable(self):
        return True

    def readline(self, limit=-1):
        raise NotImplementedError

    def readlines(self, hint=-1):
        raise NotImplementedError

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

    def truncate(self, size=None):
        raise NotImplementedError

    def writable(self):
        return False

    def writelines(self, lines):
        raise NotImplementedError

    def read(self, n=-1):
        if n == -1:
            return self.readall()

        if n <= 0:
            raise ValueError("expected n >= 1, was {}".format(n))

        if self.pos >= self.size:
            return b""

        nbytes = min(self.size - self.pos, n)

        cache_hit = self.cache_begin <= self.pos < self.cache_end

        if not cache_hit:
            readsize = max(self.cachesize, nbytes)
            download_stream = self.blob_client.download_blob(self.pos, readsize)
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

        container_client = self.blob_service.get_container_client(self.container)
        blob_client = container_client.get_blob_client(self.blobname)
        download_stream = blob_client.download_blob(self.pos)
        chunk = download_stream.readall()
        self.pos = self.size
        return chunk

    def readinto(self, b):
        # TODO optimize? Issue #253
        chunk = self.read()
        np.copyto(b, np.frombuffer(chunk, b.dtype).reshape(b.shape))
        return len(chunk)

    def write(self, b):
        raise NotImplementedError
