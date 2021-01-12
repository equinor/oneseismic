#!/usr/bin/env python3

import argparse
import contextlib
import io
import json
import numpy as np
import oneseismic
import segyio

textheader_size = 3200
header_size = 240
endian = 'big'
from oneseismic.scan.scan import format_size

nulltrace = np.zeros(0, dtype = np.float32)

import oneseismic.scan.segmenter
class padder(oneseismic.scan.segmenter.scanner):
    def __init__(self, endian, key1s, key2s, key1, key2, stream):
        super().__init__(endian)
        self.prev1 = None
        self.prev2 = None
        self.key1s = key1s
        self.key2s = key2s
        self.key1  = key1
        self.key2  = key2
        self.stream = stream

    def add(self, header):
        key1 = header[self.key1]
        key2 = header[self.key2]

        if self.prev1 != key1:
            # fill out trailing null traces for the previous line
            for key in self.key2s[self.key2s.index(self.prev2)+1:]:
                header[self.key2] = key
                self.stream.write(header.buf)
                self.stream.write(nulltrace)

            # fill out leading null traces for this line
            for key in self.key2s[:self.key2s.index(key2)]:
                header[self.key2] = key
                self.stream.write(header.buf)
                self.stream.write(nulltrace)

        self.prev1 = key1
        self.prev2 = key2

class carbonfile:
    def __init__(self, sin, sout):
        self.sin = sin
        self.sout = sout

    def seek(self, pos, whence = io.SEEK_CUR):
        if whence == io.SEEK_CUR:
            self.read(pos)

    def read(self, n=-1):
        chunk = self.sin.read(n)
        self.sout.write(chunk)
        return chunk

    @contextlib.contextmanager
    def deferred_read(self, n=-1):
        chunk = self.sin.read(n)
        yield chunk
        self.sout.write(chunk)

    def write(self, b):
        self.sout.write(b)

import struct
def swap32(i):
    return struct.unpack(">I", struct.pack("<I", i))[0]

def main(sin, sout, key1s, key2s, key1, key2, endian):
    stream = carbonfile(sin, sout)
    action = padder(
            endian = endian,
            key1s  = key1s,
            key2s  = key2s,
            key1   = key1,
            key2   = key2,
            stream = stream,
    )
    stream.seek(textheader_size, io.SEEK_CUR)
    meta = action.scan_binary(stream)

    with stream.deferred_read(header_size) as chunk:
        header = segyio.field.Field(buf = bytearray(chunk), kind = 'trace')
        header.flush = lambda _: 1
        action.scan_first_header(header)
        nulltrace.resize(action.tracelen())

        key1 = header[key1]
        key2 = header[key2]

        action.prev1 = key1
        action.prev2 = key2
        for key in action.key2s[:action.key2s.index(key2)]:
            header[action.key2] = key
            stream.write(header.buf)
            stream.write(nulltrace)

    stream.seek(len(nulltrace), io.SEEK_CUR)

    trace_count = 1
    while True:
        with stream.deferred_read(header_size) as chunk:
            if len(chunk) == 0:
                break

            if len(chunk) != header_size:
                msg = 'file truncated at trace {}'.format(trace_count)
                raise RuntimeError(msg)

            header = segyio.field.Field(buf = bytearray(chunk), kind = 'trace')
            action.add(header)

        stream.seek(len(nulltrace), io.SEEK_CUR)
        trace_count += 1

        if trace_count % 10000 == 0:
            print(f'{trace_count} processed', file = sys.stderr)

if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser(
        prog = 'pad',
        description = 'Pad SEG-Y',
    )
    parser.add_argument('src',  type = str)
    parser.add_argument('dst',  type = str)
    parser.add_argument('keys', type = str)
    parser.add_argument('--key1', type = int, default = 189)
    parser.add_argument('--key2', type = int, default = 193)
    parser.add_argument('--endian', choices = ['little', 'big'], default = 'big')
    args = parser.parse_args()

    with open(args.src, 'rb') as src, open(args.dst, 'wb') as dst:
        with open(args.keys) as k:
            keys = json.load(k)
        main(
            src,
            dst,
            key1s  = keys['dimensions'][0],
            key2s  = keys['dimensions'][1],
            key1   = args.key1,
            key2   = args.key2,
            endian = args.endian,
        )
