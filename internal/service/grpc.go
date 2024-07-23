package service

import (
	"context"

	"github.com/pentops/go-grpc-helpers/protovalidatemw"
	"github.com/pentops/log.go/grpc_log"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-auth/gen/o5/auth/v1/auth_pb"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/grpc"
)

func GRPCMiddleware() []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{
		grpc_log.UnaryServerInterceptor(log.DefaultContext, log.DefaultTrace, log.DefaultLogger),
		protovalidatemw.UnaryServerInterceptor(),
		PSMActionMiddleware(actorExtractor),
	}
}

var anonymousSubjectId = "0CBBD346-55C6-47AB-87D0-97E5C16F9DC8"

func actorExtractor(ctx context.Context) *auth_pb.Actor {
	return &auth_pb.Actor{
		AuthenticationMethod: &auth_pb.AuthenticationMethod{
			Type: &auth_pb.AuthenticationMethod_External_{
				External: &auth_pb.AuthenticationMethod_External{
					SystemName: "none",
				},
			},
		},
		SubjectId: anonymousSubjectId,
		Claim:     &auth_pb.Claim{},
	}
}

type actionContextKey struct{}

type PSMAction struct {
	Method string
	Actor  *auth_pb.Actor
}

// PSMCause is a gRPC middleware that injects the PSM cause into t he context.
func PSMActionMiddleware(actorExtractor func(context.Context) *auth_pb.Actor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		actor := actorExtractor(ctx)
		cause := PSMAction{
			Method: info.FullMethod,
			Actor:  actor,
		}
		ctx = context.WithValue(ctx, actionContextKey{}, cause)
		return handler(ctx, req)
	}
}

func WithPSMAction(ctx context.Context, action PSMAction) context.Context {
	return context.WithValue(ctx, actionContextKey{}, action)
}

func CommandCause(ctx context.Context) *psm_pb.Cause {

	cause, ok := ctx.Value(actionContextKey{}).(PSMAction)
	if !ok {
		return nil
	}

	return &psm_pb.Cause{
		Type: &psm_pb.Cause_Command{
			Command: &auth_pb.Action{
				Method: cause.Method,
				Actor:  cause.Actor,
			},
		},
	}
}
