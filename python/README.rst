oneseismic
==========
This is the root and namespace package for various modules under the oneseismic
umbrella. It includes both libraries and programs.

oneseismic.client
=================
Example

::

   from oneseismic import client

   c = client.client("endpoint")
   cube = c.cube("cube_id")
   slice = cube.slice(dim=0, lineno=1)

oneseismic.login
================
Login to oneseismic, fetch token and cache on disk.

::

   python3 -m oneseismic login

This function will prompt user to open url to provide credentials. Once this is
done, the token can be loaded from the cache and refreshed non-interactively.

oneseismic.scan
===============
Understand seismic cube identity and layout.

::

   python3 -m oneseismic scan

Scan a SEG-Y and understand its shape and layout in preparation for onseisemic
storage. The scan program writes a metadata file to be used in the next step of
the ingestion pipeline. In addition to determining the layout of data, a unique
identifier (from sha1) is computed.

The output of this program is used as input to the next step in oneseismic
upload.

For 3D inline sorted post stack volumes with standard header layout (SEG-Y rev1
or later), no options or configuration should be required. By inline sorting it
is understood that the inline is the last header word to change when reading
headers sequentially.

oneseismic scan applies the terms primary-word and secondary-word to what in
inline-sorted cubes are inline and crossline respecitvely. Consider these headers:

::

    { 189: 1, 193: 1 }
    { 189: 1, 193: 2 }
    { 189: 1, 193: 3 }
    { 189: 2, 193: 1 }
    { 189: 2, 193: 2 }
    { 189: 2, 193: 3 }

Here 189 is the primary-word and 193 is the secondary-word. If the headers were
flipped like this, then 193 would be the primary-word and 189 the
secondary-word:

::

    { 189: 1, 193: 1 }
    { 189: 2, 193: 1 }
    { 189: 3, 193: 1 }
    { 189: 1, 193: 2 }
    { 189: 2, 193: 2 }
    { 189: 3, 193: 2 }

No particular signifiance is given to either orientiation, but oneseismic-scan
requires primary-word to be the last dimension to wrap around.

oneseismic.upload
=================
Upload cubes to storage.

::

   python3 -m oneseismic upload

Upload a cube and its manifest to storage. The geometry must first be
determined and recorded with oneseismic scan.
