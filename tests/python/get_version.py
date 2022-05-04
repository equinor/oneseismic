# dummy script to keep track of the oneseismic version used
import oneseismic


def get_version():
    try:
        return oneseismic.__version__
    except:
        return 'Version unknown'


if __name__ == "__main__":
    print(get_version())
