import io


class BlobIO:
    """
    An io.RawIO like file object for azure blob store that provides a familiar
    interface for python programs
    """

    def __init__(self, blob, container, filename):
        """
        Parameters
        ----------
        blob : azure.storage.blob.*
            An azure.storage.blob BlobServiceClient object
        container : str_like
            Blob container name
        filename : str_like
            Blob file name
        Examples
        --------
        >>> from azure.storage.blob import BlobServiceClient
        >>> blob_service_client = BlobServiceClient.from_connection_string(connect_str)
        >>> stream = BlobIO(blob=blob_service_client, container="dir", filename="file.csv")
        """
        self.pos = 0
        self.blob = blob
        self.container = str(container)
        self.filename = str(filename)
        cc = blob.get_container_client(container)
        bc = cc.get_blob_client(filename)
        # TODO get total size
        self.size = bc.get_blob_properties().size

    def close(self):
        pass

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
        cc = self.blob.get_container_client(self.container)
        bc = cc.get_blob_client(self.filename)
        download_stream = bc.download_blob(self.pos, self.pos + (nbytes - 1))
        chunk = download_stream.readall()

        self.pos += len(chunk)
        return chunk

    def readall(self):
        if self.pos >= self.size:
            return b""

        cc = self.blob.get_container_client(self.container)
        bc = cc.get_blob_client(self.filename)
        download_stream = bc.download_blob(self.pos)
        chunk = download_stream.readall()
        self.pos = self.size
        return chunk

    def readinto(self, b):
        raise NotImplementedError

    def write(self, b):
        raise NotImplementedError
