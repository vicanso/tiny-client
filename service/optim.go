package service

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"

	pb "github.com/vicanso/tiny/pb"
)

type (
	// OptimParams optim params
	OptimParams struct {
		Data    []byte
		Type    string
		Quality int
	}
)

// GetConnection get connection
func GetConnection(address string) (conn *grpc.ClientConn, err error) {
	return grpc.Dial(address, grpc.WithInsecure())
}

// Optim optim image
func Optim(conn *grpc.ClientConn, params *OptimParams) (data []byte, err error) {
	c := pb.NewOptimClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	imgType := pb.Type_JPEG
	switch params.Type {
	case "png":
		imgType = pb.Type_PNG
	case "webp":
		imgType = pb.Type_WEBP
	case "jpg":
		fallthrough
	case "jpeg":
		imgType = pb.Type_JPEG
	default:
		err = errors.New("not support " + params.Type + " type")
		return
	}

	reply, err := c.DoOptim(ctx, &pb.OptimRequest{
		Source:  imgType,
		Data:    params.Data,
		Output:  imgType,
		Quality: uint32(params.Quality),
	})
	if err != nil {
		return
	}
	data = reply.Data
	return
}
