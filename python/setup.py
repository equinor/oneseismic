#!/usr/bin/env python3

import skbuild

if __name__ == "__main__":
    skbuild.setup(
        packages = [
            'oneseismic',
            'oneseismic.internal',
            'oneseismic.simple',
            'oneseismic.decoding',
            'oneseismic.scan',
            'oneseismic.upload',
        ],
    )
