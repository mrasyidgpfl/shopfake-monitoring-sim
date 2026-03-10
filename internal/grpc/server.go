package grpc

import (
	"context"
	"time"

	"github.com/mrasyidgpfl/shopfake/internal/metrics"
	"github.com/mrasyidgpfl/shopfake/internal/store"
	pb "github.com/mrasyidgpfl/shopfake/proto/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EventServer struct {
	pb.UnimplementedEventServiceServer
	store *store.PostgresStore
}

func NewEventServer(store *store.PostgresStore) *EventServer {
	return &EventServer{store: store}
}

func (s *EventServer) GetRegionStats(ctx context.Context, req *pb.RegionStatsRequest) (*pb.RegionStatsResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.GRPCRequestDuration.WithLabelValues("GetRegionStats").Observe(duration)
	}()

	stats, err := s.store.GetRegionStats(ctx)
	if err != nil {
		metrics.GRPCRequestsTotal.WithLabelValues("GetRegionStats", "error").Inc()
		return nil, status.Errorf(codes.Internal, "failed to get region stats: %v", err)
	}

	metrics.GRPCRequestsTotal.WithLabelValues("GetRegionStats", "ok").Inc()

	var pbStats []*pb.RegionStat
	for _, stat := range stats {
		pbStats = append(pbStats, &pb.RegionStat{
			Region:       string(stat.Region),
			OrderCount:   stat.OrderCount,
			FailureCount: stat.FailureCount,
			Revenue:      stat.Revenue,
			AvgLatencyMs: stat.AvgLatencyMs,
		})
	}

	return &pb.RegionStatsResponse{Stats: pbStats}, nil
}

func (s *EventServer) GetEventCounts(ctx context.Context, req *pb.EventCountsRequest) (*pb.EventCountsResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.GRPCRequestDuration.WithLabelValues("GetEventCounts").Observe(duration)
	}()

	counts, err := s.store.GetEventCountByType(ctx)
	if err != nil {
		metrics.GRPCRequestsTotal.WithLabelValues("GetEventCounts", "error").Inc()
		return nil, status.Errorf(codes.Internal, "failed to get event counts: %v", err)
	}

	metrics.GRPCRequestsTotal.WithLabelValues("GetEventCounts", "ok").Inc()

	pbCounts := make(map[string]int64)
	for k, v := range counts {
		pbCounts[string(k)] = v
	}

	return &pb.EventCountsResponse{Counts: pbCounts}, nil
}
