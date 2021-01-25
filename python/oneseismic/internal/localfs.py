import io
from pathlib import Path

class localfs:
    """Local file system

    The localfs implements a simple virtual filesystem abstraction, intended
    for testing. localfs handles path resolution, file opening and creation,
    but without modifying the environment or working directory. Absolute paths
    will be interpreted relative to root.

    Parameters
    ----------
    root : pathlib.Path or str
        Root of the virtual filesystem

    See also
    --------
    oneseismic.internal.blobfs : Azure blob file system
    """
    def __init__(self, root):
        self.root = Path(root)
        self.cwd  = Path(self.root)

    def mkdir(self, path):
        """Make directory

        Parameters
        ----------
        path : pathlib.Path or str
            Directory to make

        Notes
        -----
        This function will also succeed if the directory already exists, and
        will create parent directories if necessary.
        """
        path = Path(path)
        if path.is_absolute():
            dirpath = self.root.joinpath(path)
        else:
            dirpath = self.cwd.joinpath(path)

        dirpath.mkdir(parents = True, exist_ok = True)

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
        path = Path(path)
        if path.is_absolute():
            self.cwd = self.root.joinpath(path)
        else:
            self.cwd = self.cwd.joinpath(path)

    def open(self, name, mode = 'rb'):
        """Open file and return a stream

        Parameters
        ----------
        name : pathlib.Path or str
            File name to open
        mode : str
            Mode in which the file is opened. Identical to the builtin open()
        """
        path = self.cwd.joinpath(name)
        path.parent.mkdir(parents = True, exist_ok = True)
        return path.open(mode = mode)
