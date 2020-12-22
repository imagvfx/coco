package main

import (
	"context"
	"time"

	"github.com/imagvfx/coco"
	"github.com/imagvfx/coco/pb"
	"google.golang.org/grpc"
)

type WorkerStatus int

const (
	WorkerNotFound = WorkerStatus(iota)
	WorkerIdle
	WorkerRunning
)

type Worker struct {
	addr   string
	status WorkerStatus
}

func sendCommands(worker string, cmds []coco.Command) error {
	conn, err := grpc.Dial(worker, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		return err
	}
	defer conn.Close()

	c := pb.NewWorkerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pbCmds := &pb.Commands{}
	for _, c := range cmds {
		pbCmd := &pb.Command{
			Args: c,
		}
		pbCmds.Cmds = append(pbCmds.Cmds, pbCmd)
	}
	_, err = c.Run(ctx, pbCmds)
	if err != nil {
		return err
	}
	return nil
}
