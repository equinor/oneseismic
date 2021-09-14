oneseismic
==========
Next-generation seismic in the cloud.

Please note that oneseismic is under heavy development, and parts of this
document may be outdated.

What is oneseismic?
===================
Technically, oneseismic is an [API](https://en.wikipedia.org/wiki/API) for
reading seismic data in an easy and scalable manner. The biggest challenge with
seismic is the size of it - single surveys span from a few hundred megabytes to
tens or even hundreds of gigabytes. The standard format for storing seismic,
[SEG-Y](https://en.wikipedia.org/wiki/SEG-Y), is unfit for efficient data
extraction and querying.

The guiding design principle and focus of oneseismic is _programs first_ - it
is the idea that if you can build a solid foundation then imagining new
applications on top is fast and easy. The aim is to provide a powerful feature
that empowers developers and geoscientists so they can develop new and novel
applications, and get results faster and easier.

The best way to illustrate this is with a motivating example:

    import oneseismic
    import oneseismic.simple as simple

    cubeid = '...'
    client = simple.client('https://oneseismic.url')
    inline24    = cube.sliceByLineno(cubeid, dim = 0, lineno = 24 )().numpy()
    crossline13 = cube.sliceByLineno(cubeid, dim = 1, lineno = 13 )().numpy()
    time220     = cube.sliceByLineno(cubeid, dim = 2, lineno = 220)().numpy()

This Python program gets three slices - an inline slice, a crossline slice, and
a time slice, and makes them immediately available. This simple example only
demonstrates the fetching of arbitrary data, but we can also do something
useful with it:

    vintage1 = '...'
    vintage2 = '...'
    client = simple.client('https://oneseismic.url')
    proc1 = client.sliceByLineno(vintage1, dim = 1, lineno = 13)()
    proc2 = client.sliceByLineno(vintage2, dim = 1, lineno = 13)()
    slicev1 = proc1.numpy()
    slicev2 = proc2.numpy()

    diff = slicev2 - slicev1

This program computes the difference between the samples between two vintages
of the same field.

Notice that instead of immediately realising the data as a numpy array, this
program uses the temporaries proc1 and proc2. Creating a process will
_schedule_ the fetch, but not actually start serve the data right away. This
makes the oneseismic server process both queries in parallel.

Is oneseismic a database?
-------------------------
That depends on the definition of database. The goal of oneseismic is not to be
a universal storage solution for seismic data, but rather an efficient way to
work with and requests bits and pieces of seismic data. In that sense, it is a
database.

Why not SEG-Y?
--------------
SEG-Y was designed for data exchange, which means density, single-file and
in-band metadata are useful properties because it allows for space-efficient
and lossless transfer between parties. SEG-Y works well for this (with the
exception of rampant SEG-Y standards violations). However, SEG-Y is unfit for
modern computer programs:

1. Meta-data is interleaved with the data. That means the file can be split
   multiple places (good for tape!), but also means that the data cannot be
   contiguously copied.
2. SEG-Y is very trace-oriented, but there is no requirement that traces are
   laid out for efficient access of 3D shapes. Well-organised files are laid
   out for efficient inline access, but without an index it requires extra
   information, or a linear scan.
3. Even __if__ the file is well organised, reading a single time/depth slice or
   horizon is very time consuming.

Installation
============
Oneseismic is primarily an API, so there is no installation - the system is up
and running and be queried at any time. The Python
[package](https://pypi.org/project/oneseismic/) is a user-friendly way to use
the API, and can be installed with:

    pip install oneseismic

However, the API is perfectly usable without going through the Python package.
Please note that it is still under heavy development, and may change with
little notice.

Examples
========

Developer's corner
==================
This section is for the developers of oneseismic, and describes the
architecture and design choices that power oneseismic.

Offline partitioning
--------------------
When uploaded, the volume is partitioned into equally-sized chunks, which are
then addressed by its coordinates in this coarser grid. This process is time
consuming, but is only performed once. With a unique address
`<volume>/<resolution>/<partioning>/<chunk>`, which can be easily computed from
any coordinate, oneseismic can efficiently get arbitrary shapes from large
volumes.

Terminology
-----------
A lot of the familiar terminology in oneseismic is lifted from the unix family
of operating systems, since the concepts in use in oneseismic map pretty well
onto the concepts in unix. This is an incomplete list of terms used throughout
code and documentation:

process
: The process is the high-level procedure from a user request until data is
  delivered.

PID
: The PID, __process identifier__, is the key used to identify a single process
  across the sub systems. Please note that unlike traditional unix systems,
  this is represented as a string, and not a single integer.

A day in the life (of a request)
--------------------------------

License
=======
The server is licensed under AGPL v3+, while the connector and python libraries
are licensed under the LGPL v3+.
