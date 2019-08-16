// Generated by the gRPC C++ plugin.
// If you make any local change, they will be lost.
// source: core.proto
#ifndef GRPC_core_2eproto__INCLUDED
#define GRPC_core_2eproto__INCLUDED

#include "core.pb.h"

#include <functional>
#include <grpcpp/impl/codegen/async_generic_service.h>
#include <grpcpp/impl/codegen/async_stream.h>
#include <grpcpp/impl/codegen/async_unary_call.h>
#include <grpcpp/impl/codegen/client_callback.h>
#include <grpcpp/impl/codegen/method_handler_impl.h>
#include <grpcpp/impl/codegen/proto_utils.h>
#include <grpcpp/impl/codegen/rpc_method.h>
#include <grpcpp/impl/codegen/server_callback.h>
#include <grpcpp/impl/codegen/service_type.h>
#include <grpcpp/impl/codegen/status.h>
#include <grpcpp/impl/codegen/stub_options.h>
#include <grpcpp/impl/codegen/sync_stream.h>

namespace grpc {
class CompletionQueue;
class Channel;
class ServerCompletionQueue;
class ServerContext;
}  // namespace grpc

namespace seismic_core {

// The seismic core service definition.
class Core final {
 public:
  static constexpr char const* service_full_name() {
    return "seismic_core.Core";
  }
  class StubInterface {
   public:
    virtual ~StubInterface() {}
    virtual ::grpc::Status ShatterLink(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest& request, ::seismic_core::ShatterReply* response) = 0;
    std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::ShatterReply>> AsyncShatterLink(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::ShatterReply>>(AsyncShatterLinkRaw(context, request, cq));
    }
    std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::ShatterReply>> PrepareAsyncShatterLink(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::ShatterReply>>(PrepareAsyncShatterLinkRaw(context, request, cq));
    }
    virtual ::grpc::Status StitchSurface(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest& request, ::seismic_core::SurfaceReply* response) = 0;
    std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::SurfaceReply>> AsyncStitchSurface(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::SurfaceReply>>(AsyncStitchSurfaceRaw(context, request, cq));
    }
    std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::SurfaceReply>> PrepareAsyncStitchSurface(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::SurfaceReply>>(PrepareAsyncStitchSurfaceRaw(context, request, cq));
    }
    class experimental_async_interface {
     public:
      virtual ~experimental_async_interface() {}
      virtual void ShatterLink(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest* request, ::seismic_core::ShatterReply* response, std::function<void(::grpc::Status)>) = 0;
      virtual void ShatterLink(::grpc::ClientContext* context, const ::grpc::ByteBuffer* request, ::seismic_core::ShatterReply* response, std::function<void(::grpc::Status)>) = 0;
      virtual void StitchSurface(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest* request, ::seismic_core::SurfaceReply* response, std::function<void(::grpc::Status)>) = 0;
      virtual void StitchSurface(::grpc::ClientContext* context, const ::grpc::ByteBuffer* request, ::seismic_core::SurfaceReply* response, std::function<void(::grpc::Status)>) = 0;
    };
    virtual class experimental_async_interface* experimental_async() { return nullptr; }
  private:
    virtual ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::ShatterReply>* AsyncShatterLinkRaw(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest& request, ::grpc::CompletionQueue* cq) = 0;
    virtual ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::ShatterReply>* PrepareAsyncShatterLinkRaw(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest& request, ::grpc::CompletionQueue* cq) = 0;
    virtual ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::SurfaceReply>* AsyncStitchSurfaceRaw(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest& request, ::grpc::CompletionQueue* cq) = 0;
    virtual ::grpc::ClientAsyncResponseReaderInterface< ::seismic_core::SurfaceReply>* PrepareAsyncStitchSurfaceRaw(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest& request, ::grpc::CompletionQueue* cq) = 0;
  };
  class Stub final : public StubInterface {
   public:
    Stub(const std::shared_ptr< ::grpc::ChannelInterface>& channel);
    ::grpc::Status ShatterLink(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest& request, ::seismic_core::ShatterReply* response) override;
    std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::seismic_core::ShatterReply>> AsyncShatterLink(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::seismic_core::ShatterReply>>(AsyncShatterLinkRaw(context, request, cq));
    }
    std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::seismic_core::ShatterReply>> PrepareAsyncShatterLink(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::seismic_core::ShatterReply>>(PrepareAsyncShatterLinkRaw(context, request, cq));
    }
    ::grpc::Status StitchSurface(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest& request, ::seismic_core::SurfaceReply* response) override;
    std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::seismic_core::SurfaceReply>> AsyncStitchSurface(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::seismic_core::SurfaceReply>>(AsyncStitchSurfaceRaw(context, request, cq));
    }
    std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::seismic_core::SurfaceReply>> PrepareAsyncStitchSurface(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::seismic_core::SurfaceReply>>(PrepareAsyncStitchSurfaceRaw(context, request, cq));
    }
    class experimental_async final :
      public StubInterface::experimental_async_interface {
     public:
      void ShatterLink(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest* request, ::seismic_core::ShatterReply* response, std::function<void(::grpc::Status)>) override;
      void ShatterLink(::grpc::ClientContext* context, const ::grpc::ByteBuffer* request, ::seismic_core::ShatterReply* response, std::function<void(::grpc::Status)>) override;
      void StitchSurface(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest* request, ::seismic_core::SurfaceReply* response, std::function<void(::grpc::Status)>) override;
      void StitchSurface(::grpc::ClientContext* context, const ::grpc::ByteBuffer* request, ::seismic_core::SurfaceReply* response, std::function<void(::grpc::Status)>) override;
     private:
      friend class Stub;
      explicit experimental_async(Stub* stub): stub_(stub) { }
      Stub* stub() { return stub_; }
      Stub* stub_;
    };
    class experimental_async_interface* experimental_async() override { return &async_stub_; }

   private:
    std::shared_ptr< ::grpc::ChannelInterface> channel_;
    class experimental_async async_stub_{this};
    ::grpc::ClientAsyncResponseReader< ::seismic_core::ShatterReply>* AsyncShatterLinkRaw(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest& request, ::grpc::CompletionQueue* cq) override;
    ::grpc::ClientAsyncResponseReader< ::seismic_core::ShatterReply>* PrepareAsyncShatterLinkRaw(::grpc::ClientContext* context, const ::seismic_core::ShatterLinkRequest& request, ::grpc::CompletionQueue* cq) override;
    ::grpc::ClientAsyncResponseReader< ::seismic_core::SurfaceReply>* AsyncStitchSurfaceRaw(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest& request, ::grpc::CompletionQueue* cq) override;
    ::grpc::ClientAsyncResponseReader< ::seismic_core::SurfaceReply>* PrepareAsyncStitchSurfaceRaw(::grpc::ClientContext* context, const ::seismic_core::SurfaceRequest& request, ::grpc::CompletionQueue* cq) override;
    const ::grpc::internal::RpcMethod rpcmethod_ShatterLink_;
    const ::grpc::internal::RpcMethod rpcmethod_StitchSurface_;
  };
  static std::unique_ptr<Stub> NewStub(const std::shared_ptr< ::grpc::ChannelInterface>& channel, const ::grpc::StubOptions& options = ::grpc::StubOptions());

  class Service : public ::grpc::Service {
   public:
    Service();
    virtual ~Service();
    virtual ::grpc::Status ShatterLink(::grpc::ServerContext* context, const ::seismic_core::ShatterLinkRequest* request, ::seismic_core::ShatterReply* response);
    virtual ::grpc::Status StitchSurface(::grpc::ServerContext* context, const ::seismic_core::SurfaceRequest* request, ::seismic_core::SurfaceReply* response);
  };
  template <class BaseClass>
  class WithAsyncMethod_ShatterLink : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    WithAsyncMethod_ShatterLink() {
      ::grpc::Service::MarkMethodAsync(0);
    }
    ~WithAsyncMethod_ShatterLink() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status ShatterLink(::grpc::ServerContext* context, const ::seismic_core::ShatterLinkRequest* request, ::seismic_core::ShatterReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    void RequestShatterLink(::grpc::ServerContext* context, ::seismic_core::ShatterLinkRequest* request, ::grpc::ServerAsyncResponseWriter< ::seismic_core::ShatterReply>* response, ::grpc::CompletionQueue* new_call_cq, ::grpc::ServerCompletionQueue* notification_cq, void *tag) {
      ::grpc::Service::RequestAsyncUnary(0, context, request, response, new_call_cq, notification_cq, tag);
    }
  };
  template <class BaseClass>
  class WithAsyncMethod_StitchSurface : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    WithAsyncMethod_StitchSurface() {
      ::grpc::Service::MarkMethodAsync(1);
    }
    ~WithAsyncMethod_StitchSurface() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status StitchSurface(::grpc::ServerContext* context, const ::seismic_core::SurfaceRequest* request, ::seismic_core::SurfaceReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    void RequestStitchSurface(::grpc::ServerContext* context, ::seismic_core::SurfaceRequest* request, ::grpc::ServerAsyncResponseWriter< ::seismic_core::SurfaceReply>* response, ::grpc::CompletionQueue* new_call_cq, ::grpc::ServerCompletionQueue* notification_cq, void *tag) {
      ::grpc::Service::RequestAsyncUnary(1, context, request, response, new_call_cq, notification_cq, tag);
    }
  };
  typedef WithAsyncMethod_ShatterLink<WithAsyncMethod_StitchSurface<Service > > AsyncService;
  template <class BaseClass>
  class ExperimentalWithCallbackMethod_ShatterLink : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    ExperimentalWithCallbackMethod_ShatterLink() {
      ::grpc::Service::experimental().MarkMethodCallback(0,
        new ::grpc::internal::CallbackUnaryHandler< ::seismic_core::ShatterLinkRequest, ::seismic_core::ShatterReply>(
          [this](::grpc::ServerContext* context,
                 const ::seismic_core::ShatterLinkRequest* request,
                 ::seismic_core::ShatterReply* response,
                 ::grpc::experimental::ServerCallbackRpcController* controller) {
                   return this->ShatterLink(context, request, response, controller);
                 }));
    }
    ~ExperimentalWithCallbackMethod_ShatterLink() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status ShatterLink(::grpc::ServerContext* context, const ::seismic_core::ShatterLinkRequest* request, ::seismic_core::ShatterReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    virtual void ShatterLink(::grpc::ServerContext* context, const ::seismic_core::ShatterLinkRequest* request, ::seismic_core::ShatterReply* response, ::grpc::experimental::ServerCallbackRpcController* controller) { controller->Finish(::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "")); }
  };
  template <class BaseClass>
  class ExperimentalWithCallbackMethod_StitchSurface : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    ExperimentalWithCallbackMethod_StitchSurface() {
      ::grpc::Service::experimental().MarkMethodCallback(1,
        new ::grpc::internal::CallbackUnaryHandler< ::seismic_core::SurfaceRequest, ::seismic_core::SurfaceReply>(
          [this](::grpc::ServerContext* context,
                 const ::seismic_core::SurfaceRequest* request,
                 ::seismic_core::SurfaceReply* response,
                 ::grpc::experimental::ServerCallbackRpcController* controller) {
                   return this->StitchSurface(context, request, response, controller);
                 }));
    }
    ~ExperimentalWithCallbackMethod_StitchSurface() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status StitchSurface(::grpc::ServerContext* context, const ::seismic_core::SurfaceRequest* request, ::seismic_core::SurfaceReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    virtual void StitchSurface(::grpc::ServerContext* context, const ::seismic_core::SurfaceRequest* request, ::seismic_core::SurfaceReply* response, ::grpc::experimental::ServerCallbackRpcController* controller) { controller->Finish(::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "")); }
  };
  typedef ExperimentalWithCallbackMethod_ShatterLink<ExperimentalWithCallbackMethod_StitchSurface<Service > > ExperimentalCallbackService;
  template <class BaseClass>
  class WithGenericMethod_ShatterLink : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    WithGenericMethod_ShatterLink() {
      ::grpc::Service::MarkMethodGeneric(0);
    }
    ~WithGenericMethod_ShatterLink() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status ShatterLink(::grpc::ServerContext* context, const ::seismic_core::ShatterLinkRequest* request, ::seismic_core::ShatterReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
  };
  template <class BaseClass>
  class WithGenericMethod_StitchSurface : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    WithGenericMethod_StitchSurface() {
      ::grpc::Service::MarkMethodGeneric(1);
    }
    ~WithGenericMethod_StitchSurface() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status StitchSurface(::grpc::ServerContext* context, const ::seismic_core::SurfaceRequest* request, ::seismic_core::SurfaceReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
  };
  template <class BaseClass>
  class WithRawMethod_ShatterLink : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    WithRawMethod_ShatterLink() {
      ::grpc::Service::MarkMethodRaw(0);
    }
    ~WithRawMethod_ShatterLink() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status ShatterLink(::grpc::ServerContext* context, const ::seismic_core::ShatterLinkRequest* request, ::seismic_core::ShatterReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    void RequestShatterLink(::grpc::ServerContext* context, ::grpc::ByteBuffer* request, ::grpc::ServerAsyncResponseWriter< ::grpc::ByteBuffer>* response, ::grpc::CompletionQueue* new_call_cq, ::grpc::ServerCompletionQueue* notification_cq, void *tag) {
      ::grpc::Service::RequestAsyncUnary(0, context, request, response, new_call_cq, notification_cq, tag);
    }
  };
  template <class BaseClass>
  class WithRawMethod_StitchSurface : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    WithRawMethod_StitchSurface() {
      ::grpc::Service::MarkMethodRaw(1);
    }
    ~WithRawMethod_StitchSurface() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status StitchSurface(::grpc::ServerContext* context, const ::seismic_core::SurfaceRequest* request, ::seismic_core::SurfaceReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    void RequestStitchSurface(::grpc::ServerContext* context, ::grpc::ByteBuffer* request, ::grpc::ServerAsyncResponseWriter< ::grpc::ByteBuffer>* response, ::grpc::CompletionQueue* new_call_cq, ::grpc::ServerCompletionQueue* notification_cq, void *tag) {
      ::grpc::Service::RequestAsyncUnary(1, context, request, response, new_call_cq, notification_cq, tag);
    }
  };
  template <class BaseClass>
  class ExperimentalWithRawCallbackMethod_ShatterLink : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    ExperimentalWithRawCallbackMethod_ShatterLink() {
      ::grpc::Service::experimental().MarkMethodRawCallback(0,
        new ::grpc::internal::CallbackUnaryHandler< ::grpc::ByteBuffer, ::grpc::ByteBuffer>(
          [this](::grpc::ServerContext* context,
                 const ::grpc::ByteBuffer* request,
                 ::grpc::ByteBuffer* response,
                 ::grpc::experimental::ServerCallbackRpcController* controller) {
                   this->ShatterLink(context, request, response, controller);
                 }));
    }
    ~ExperimentalWithRawCallbackMethod_ShatterLink() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status ShatterLink(::grpc::ServerContext* context, const ::seismic_core::ShatterLinkRequest* request, ::seismic_core::ShatterReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    virtual void ShatterLink(::grpc::ServerContext* context, const ::grpc::ByteBuffer* request, ::grpc::ByteBuffer* response, ::grpc::experimental::ServerCallbackRpcController* controller) { controller->Finish(::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "")); }
  };
  template <class BaseClass>
  class ExperimentalWithRawCallbackMethod_StitchSurface : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    ExperimentalWithRawCallbackMethod_StitchSurface() {
      ::grpc::Service::experimental().MarkMethodRawCallback(1,
        new ::grpc::internal::CallbackUnaryHandler< ::grpc::ByteBuffer, ::grpc::ByteBuffer>(
          [this](::grpc::ServerContext* context,
                 const ::grpc::ByteBuffer* request,
                 ::grpc::ByteBuffer* response,
                 ::grpc::experimental::ServerCallbackRpcController* controller) {
                   this->StitchSurface(context, request, response, controller);
                 }));
    }
    ~ExperimentalWithRawCallbackMethod_StitchSurface() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status StitchSurface(::grpc::ServerContext* context, const ::seismic_core::SurfaceRequest* request, ::seismic_core::SurfaceReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    virtual void StitchSurface(::grpc::ServerContext* context, const ::grpc::ByteBuffer* request, ::grpc::ByteBuffer* response, ::grpc::experimental::ServerCallbackRpcController* controller) { controller->Finish(::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "")); }
  };
  template <class BaseClass>
  class WithStreamedUnaryMethod_ShatterLink : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    WithStreamedUnaryMethod_ShatterLink() {
      ::grpc::Service::MarkMethodStreamed(0,
        new ::grpc::internal::StreamedUnaryHandler< ::seismic_core::ShatterLinkRequest, ::seismic_core::ShatterReply>(std::bind(&WithStreamedUnaryMethod_ShatterLink<BaseClass>::StreamedShatterLink, this, std::placeholders::_1, std::placeholders::_2)));
    }
    ~WithStreamedUnaryMethod_ShatterLink() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable regular version of this method
    ::grpc::Status ShatterLink(::grpc::ServerContext* context, const ::seismic_core::ShatterLinkRequest* request, ::seismic_core::ShatterReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    // replace default version of method with streamed unary
    virtual ::grpc::Status StreamedShatterLink(::grpc::ServerContext* context, ::grpc::ServerUnaryStreamer< ::seismic_core::ShatterLinkRequest,::seismic_core::ShatterReply>* server_unary_streamer) = 0;
  };
  template <class BaseClass>
  class WithStreamedUnaryMethod_StitchSurface : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service *service) {}
   public:
    WithStreamedUnaryMethod_StitchSurface() {
      ::grpc::Service::MarkMethodStreamed(1,
        new ::grpc::internal::StreamedUnaryHandler< ::seismic_core::SurfaceRequest, ::seismic_core::SurfaceReply>(std::bind(&WithStreamedUnaryMethod_StitchSurface<BaseClass>::StreamedStitchSurface, this, std::placeholders::_1, std::placeholders::_2)));
    }
    ~WithStreamedUnaryMethod_StitchSurface() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable regular version of this method
    ::grpc::Status StitchSurface(::grpc::ServerContext* context, const ::seismic_core::SurfaceRequest* request, ::seismic_core::SurfaceReply* response) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    // replace default version of method with streamed unary
    virtual ::grpc::Status StreamedStitchSurface(::grpc::ServerContext* context, ::grpc::ServerUnaryStreamer< ::seismic_core::SurfaceRequest,::seismic_core::SurfaceReply>* server_unary_streamer) = 0;
  };
  typedef WithStreamedUnaryMethod_ShatterLink<WithStreamedUnaryMethod_StitchSurface<Service > > StreamedUnaryService;
  typedef Service SplitStreamedService;
  typedef WithStreamedUnaryMethod_ShatterLink<WithStreamedUnaryMethod_StitchSurface<Service > > StreamedService;
};

}  // namespace seismic_core


#endif  // GRPC_core_2eproto__INCLUDED
