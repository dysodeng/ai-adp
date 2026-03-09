package errors

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Pipeline 使用 errgroup 的链式管道多错误处理
type Pipeline struct {
	ctx        context.Context
	g          *errgroup.Group
	fns        []func() error
	finalizers []func()
}

func NewPipeline() *Pipeline {
	return NewPipelineWithContext(context.Background())
}

func NewPipelineWithContext(ctx context.Context) *Pipeline {
	g, ctx := errgroup.WithContext(ctx)
	return &Pipeline{
		ctx:        ctx,
		g:          g,
		fns:        make([]func() error, 0),
		finalizers: make([]func(), 0),
	}
}

// Then 添加一个函数到管道中
func (p *Pipeline) Then(fn ...func() error) *Pipeline {
	p.fns = append(p.fns, fn...)
	return p
}

// Finally 注册一个收尾回调，无论管道成功或失败都会执行一次
func (p *Pipeline) Finally(fn ...func()) *Pipeline {
	p.finalizers = append(p.finalizers, fn...)
	return p
}

func (p *Pipeline) runFinally() {
	for _, f := range p.finalizers {
		f()
	}
}

// Execute 顺序执行所有函数（遇到错误立即停止）
func (p *Pipeline) Execute() error {
	defer p.runFinally()

	for _, fn := range p.fns {
		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		default:
		}

		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteParallel 并发执行所有函数
func (p *Pipeline) ExecuteParallel() error {
	defer p.runFinally()

	for _, fn := range p.fns {
		fn := fn // 避免闭包问题
		p.g.Go(func() error {
			return fn()
		})
	}

	return p.g.Wait()
}

// ExecuteParallelWithLimit 限制并发数量执行
func (p *Pipeline) ExecuteParallelWithLimit(limit int) error {
	defer p.runFinally()

	g := new(errgroup.Group)
	g.SetLimit(limit)

	for _, fn := range p.fns {
		fn := fn // 避免闭包问题
		g.Go(func() error {
			return fn()
		})
	}

	return g.Wait()
}

// Context 获取上下文
func (p *Pipeline) Context() context.Context {
	return p.ctx
}
