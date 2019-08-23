from concurrent import futures
import logging
import sys
import time

import grpc
from . import stitch

from .proto import core_pb2
from .proto import core_pb2_grpc

_ONE_DAY_IN_SECONDS = 60 * 60 * 24

class Core(core_pb2_grpc.CoreServicer):

    def __init__(self, base_dir):
        self.base_dir = base_dir

    def ShatterLink(self, request, context):
        return core_pb2.ShatterReply(message='Got link %s!' % request.link)

    def StitchSurface(self, request, context):
        reply = core_pb2.SurfaceReply()
        stitch.surface(self.base_dir, request.surface, request.basename,
                       request.cubexs, request.cubeys, request.cubezs,
                       request.fragmentxs, request.fragmentys, request.fragmentzs,
                       reply)
        return reply


def serve(base_dir):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    core_pb2_grpc.add_CoreServicer_to_server(Core(base_dir), server)
    server.add_insecure_port('localhost:5005')
    server.start()
    try:
        while True:
            time.sleep(_ONE_DAY_IN_SECONDS)
    except KeyboardInterrupt:
        server.stop(0)



def main():
    logging.basicConfig()
    serve(sys.argv[1])

if __name__ == '__main__':
    main()
