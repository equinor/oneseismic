import sys
import json
import matplotlib.pyplot as plt
import numpy as np
np.set_printoptions(threshold=sys.maxsize)

import grpc

import protos.core_pb2_grpc
from protos.core_pb2 import Geometry, SliceRequest

def main(manifest, dim, lineno):
    channel = grpc.insecure_channel('localhost:50051',
        options = [
            ('grpc.max_receive_message_length', -1),
        ])
    stub = protos.core_pb2_grpc.CoreStub(channel)

    with open(manifest, 'r') as f:
        manifest = json.load(f)

    dim2 = [int(x * 1000) for x in manifest['dimensions'][2]]
    geo = Geometry(
        guid = manifest['guid'],
        dim0 = manifest['dimensions'][0],
        dim1 = manifest['dimensions'][1],
        dim2 = dim2,
    )

    dim = {
        'inline': 0,
        'crossline': 1,
        'depth': 2,
    }[dim]

    import timeit

    start_time = timeit.default_timer()
    request = SliceRequest(
        dim = dim,
        lineno =lineno,
        dim0 = 64,
        dim1 = 64,
        dim2 = 64,
        geometry = geo,
    )
    reply = stub.Slice(request)
    elapsed = timeit.default_timer() - start_time
    print('request took {0:.3f}s'.format(elapsed))

    cube = np.array([v for v in reply.v])
    cube = cube.reshape((reply.dim0, reply.dim1))
    # plt.imshow(cube.T, cmap=plt.get_cmap('gray'), vmin = -1, vmax = 1)
    plt.imshow(cube.T, cmap=plt.get_cmap('seismic'))
    plt.show()

if __name__ == '__main__':
    main(sys.argv[1], sys.argv[2], int(sys.argv[3]))
