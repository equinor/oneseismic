# Generated by the gRPC Python protocol compiler plugin. DO NOT EDIT!
import grpc

from proto import core_pb2 as core__pb2


class CoreStub(object):
  """The seismic core service definition.
  """

  def __init__(self, channel):
    """Constructor.

    Args:
      channel: A grpc.Channel.
    """
    self.ShatterLink = channel.unary_unary(
        '/seismic_core.Core/ShatterLink',
        request_serializer=core__pb2.ShatterLinkRequest.SerializeToString,
        response_deserializer=core__pb2.ShatterReply.FromString,
        )
    self.StitchSurface = channel.unary_unary(
        '/seismic_core.Core/StitchSurface',
        request_serializer=core__pb2.SurfaceRequest.SerializeToString,
        response_deserializer=core__pb2.SurfaceReply.FromString,
        )


class CoreServicer(object):
  """The seismic core service definition.
  """

  def ShatterLink(self, request, context):
    # missing associated documentation comment in .proto file
    pass
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')

  def StitchSurface(self, request, context):
    # missing associated documentation comment in .proto file
    pass
    context.set_code(grpc.StatusCode.UNIMPLEMENTED)
    context.set_details('Method not implemented!')
    raise NotImplementedError('Method not implemented!')


def add_CoreServicer_to_server(servicer, server):
  rpc_method_handlers = {
      'ShatterLink': grpc.unary_unary_rpc_method_handler(
          servicer.ShatterLink,
          request_deserializer=core__pb2.ShatterLinkRequest.FromString,
          response_serializer=core__pb2.ShatterReply.SerializeToString,
      ),
      'StitchSurface': grpc.unary_unary_rpc_method_handler(
          servicer.StitchSurface,
          request_deserializer=core__pb2.SurfaceRequest.FromString,
          response_serializer=core__pb2.SurfaceReply.SerializeToString,
      ),
  }
  generic_handler = grpc.method_handlers_generic_handler(
      'seismic_core.Core', rpc_method_handlers)
  server.add_generic_rpc_handlers((generic_handler,))
