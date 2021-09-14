#!/usr/bin/env python3

import skbuild

if __name__ == "__main__":
    skbuild.setup(
        packages = [
            'oneseismic',
            'oneseismic.client',
            'oneseismic.internal',
            'oneseismic.scan',
            'oneseismic.upload',
        ],
    )
