package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/regentmarkets/service-pricer-digitalcallput/api/digitalcallput"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/feed"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/grpcsvc"
	"github.com/regentmarkets/service-pricer-digitalcallput/internal/pricer"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Initialize dependencies
	p := pricer.NewPricer()
	f := feed.NewMockFeedClient() // Use mock for now
	svc := grpcsvc.NewServer(p, f)

	// Create gRPC server
	s := grpc.NewServer()
	pb.RegisterDigitalcallputServiceServer(s, svc)

	// Enable reflection for debugging
	reflection.Register(s)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
