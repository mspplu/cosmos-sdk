package keeper

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/ibc/03-connection/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
)

var _ types.QueryServer = Keeper{}

// Connection implements the Query/Connection gRPC method
func (q Keeper) Connection(c context.Context, req *types.QueryConnectionRequest) (*types.QueryConnectionResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if err := host.ConnectionIdentifierValidator(req.ConnectionID); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	connection, found := q.GetConnection(ctx, req.ConnectionID)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			sdkerrors.Wrap(types.ErrConnectionNotFound, req.ConnectionID).Error(),
		)
	}

	return &types.QueryConnectionResponse{
		Connection:  &connection,
		ProofHeight: uint64(ctx.BlockHeight()),
	}, nil
}

// Connections implements the Query/Connections gRPC method
func (q Keeper) Connections(c context.Context, req *types.QueryConnectionsRequest) (*types.QueryConnectionsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	connections := []*types.ConnectionEnd{}
	store := prefix.NewStore(ctx.KVStore(q.storeKey), host.KeyConnectionPrefix)

	res, err := query.Paginate(store, req.Req, func(key []byte, value []byte) error {
		var result types.ConnectionEnd
		if err := q.cdc.UnmarshalBinaryBare(value, &result); err != nil {
			return err
		}

		connections = append(connections, &result)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &types.QueryConnectionsResponse{
		Connections: connections,
		Res:         res,
		Height:      ctx.BlockHeight(),
	}, nil
}

// ClientConnections implements the Query/ClientConnections gRPC method
func (q Keeper) ClientConnections(c context.Context, req *types.QueryClientConnectionsRequest) (*types.QueryClientConnectionsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if err := host.ClientIdentifierValidator(req.ClientID); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	clientConnectionPaths, found := q.GetClientConnectionPaths(ctx, req.ClientID)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			sdkerrors.Wrap(types.ErrClientConnectionPathsNotFound, req.ClientID).Error(),
		)
	}

	return &types.QueryClientConnectionsResponse{
		ConnectionPaths: clientConnectionPaths,
		ProofHeight:     uint64(ctx.BlockHeight()),
	}, nil
}

// ClientsConnections implements the Query/ClientsConnections gRPC method
func (q Keeper) ClientsConnections(c context.Context, req *types.QueryClientsConnectionsRequest) (*types.QueryClientsConnectionsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	connectionsPaths := []*types.ConnectionPaths{}
	store := prefix.NewStore(ctx.KVStore(q.storeKey), host.KeyClientStorePrefix)

	res, err := query.Paginate(store, req.Req, func(key []byte, _ []byte) error {
		keySplit := strings.Split(string(key), "/")
		if keySplit[len(keySplit)-1] != "clientState" {
			// continue if key is not from client state
			return nil
		}

		clientStateID := keySplit[1]
		paths, found := q.GetClientConnectionPaths(ctx, clientStateID)
		if !found {
			// continue when connection handshake is not initialized
			return nil
		}

		connPaths := types.NewConnectionPaths(clientStateID, paths)
		connectionsPaths = append(connectionsPaths, &connPaths)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &types.QueryClientsConnectionsResponse{
		ConnectionsPaths: connectionsPaths,
		Res:              res,
		Height:           ctx.BlockHeight(),
	}, nil
}
