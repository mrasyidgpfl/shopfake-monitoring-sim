package grpc

import (
	"context"
	"fmt"
	"time"

	pb "github.com/mrasyidgpfl/shopfake/proto/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type EventClient struct {
	conn   *grpc.ClientConn
	client pb.EventServiceClient
}

func NewEventClient(addr string) (*EventClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to grpc server: %w", err)
	}

	return &EventClient{
		conn:   conn,
		client: pb.NewEventServiceClient(conn),
	}, nil
}

func (c *EventClient) GetRegionStats(ctx context.Context) ([]*pb.RegionStat, error) {
	resp, err := c.client.GetRegionStats(ctx, &pb.RegionStatsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get region stats: %w", err)
	}
	return resp.Stats, nil
}

func (c *EventClient) GetEventCounts(ctx context.Context) (map[string]int64, error) {
	resp, err := c.client.GetEventCounts(ctx, &pb.EventCountsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get event counts: %w", err)
	}
	return resp.Counts, nil
}

func (c *EventClient) Close() error {
	return c.conn.Close()
}
