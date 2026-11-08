package workerpool

import "context"

func Run(ctx context.Context, n int, jobs []func(context.Context) error) error {

	return ctx.Err()
}

// func worker(ctx context.Context) {

// 	for i := 0; i < 1000; i++ {
// 		_ = i * i
// 	}
// }
