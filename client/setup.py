import setuptools

setuptools.setup(
    name = 'oneseismic-client',

    packages = ['oneseismic'],

    description = 'oneseismic - Client for accessing the oneseismic API',
    url = 'https://github.com/equinor/oneseismic',
    license = 'AGPL-3.0',
    version = '0.0.0',

    entry_points = {
        'console_scripts': [
            'oneseismic-login = oneseismic.login:main',
        ]
    },

    install_requires = [
        'msal',
        'numpy',
        'requests',
        'ujson',
    ],

    setup_requires = [
        'pytest-runner',
    ],

    tests_require = [
        'pytest',
        'requests_mock',
    ],
)
