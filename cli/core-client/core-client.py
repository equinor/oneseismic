import grpc

from proto import core_pb2
from proto import core_pb2_grpc

channel = grpc.insecure_channel('localhost:5005', options=[('grpc.max_receive_message_length', 500 * 1024 * 1024)])
stub = core_pb2_grpc.CoreStub(channel)

surface_request = core_pb2.SurfaceRequest(surface='surface1.i32', basename='cube', cubexs=4000, cubeys=3000, cubezs=1500, fragmentxs=300, fragmentys=300, fragmentzs=300)

surface_reply = stub.StitchSurface(surface_request)

print(len(surface_reply.i))
v = sorted(surface_reply.v )

print(v[:100])
