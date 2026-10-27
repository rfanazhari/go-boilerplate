package runtime

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	appevent "github.com/rfanazhari/go-boilerplate/application/event"
	"github.com/rfanazhari/go-boilerplate/application/usecase"
	httpapi "github.com/rfanazhari/go-boilerplate/delivery/http"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/infrastructure/config"
	appi18n "github.com/rfanazhari/go-boilerplate/infrastructure/i18n"
	accountrepository "github.com/rfanazhari/go-boilerplate/infrastructure/repository/accountrepository"
	taskrepository "github.com/rfanazhari/go-boilerplate/infrastructure/repository/taskrepository"
	tokenrepository "github.com/rfanazhari/go-boilerplate/infrastructure/repository/tokenrepository"
	"github.com/rfanazhari/go-boilerplate/infrastructure/seed"
	infsvc "github.com/rfanazhari/go-boilerplate/infrastructure/service"
	"github.com/rfanazhari/go-boilerplate/translations"
)

type Runtime struct {
	HTTP     *http.Server
	Pool     *pgxpool.Pool
	Rabbit   infsvc.RabbitPublisher
	Shutdown time.Duration
}

func Start(ctx context.Context, cfg config.Config, logApp *logmanager.Application) (*Runtime, error) {
	bundle, err := appi18n.Load(translations.Files)
	if err != nil {
		return nil, err
	}

	var pool *pgxpool.Pool
	var tasks repository.TaskRepository = taskrepository.Unavailable{}
	var accounts repository.AccountRepository = &accountrepository.Memory{}
	var tokens repository.TokenRepository = &tokenrepository.Memory{}
	var publisher appevent.CompletionPublisher = infsvc.UnavailablePublisher{}
	var rabbit infsvc.RabbitPublisher
	if cfg.InfraOptional {
		tasks = &taskrepository.Memory{}
		publisher = &infsvc.RecordingPublisher{}
	}

	if cfg.DatabaseURL != "" {
		opened, err := pgxpool.New(ctx, cfg.DatabaseURL)
		if err != nil {
			if !cfg.InfraOptional {
				return nil, err
			}
		} else if err := opened.Ping(ctx); err != nil {
			opened.Close()
			if !cfg.InfraOptional {
				return nil, err
			}
		} else {
			pool = opened
			if err := seed.SeedAdmin(ctx, pool, cfg); err != nil {
				opened.Close()
				return nil, err
			}
			tasks = taskrepository.Postgres{Pool: pool}
			accounts = accountrepository.Postgres{Pool: pool}
			tokens = tokenrepository.Postgres{Pool: pool}
		}
	} else if !cfg.InfraOptional {
		return nil, errors.New("DATABASE_URL is required")
	}

	if cfg.RabbitURL != "" {
		conn, err := amqp.Dial(cfg.RabbitURL)
		if err != nil {
			if pool != nil {
				pool.Close()
			}
			if !cfg.InfraOptional {
				return nil, err
			}
		} else {
			ch, err := conn.Channel()
			if err != nil {
				_ = conn.Close()
				if pool != nil {
					pool.Close()
				}
				return nil, err
			}
			if err := infsvc.DeclareTaskCompleted(ch, cfg.RabbitExchange, cfg.RabbitQueue); err != nil {
				_ = ch.Close()
				_ = conn.Close()
				if pool != nil {
					pool.Close()
				}
				if !cfg.InfraOptional {
					return nil, err
				}
			} else {
				rabbit = infsvc.RabbitPublisher{Conn: conn, Channel: ch, Exchange: cfg.RabbitExchange, Queue: cfg.RabbitQueue}
				publisher = rabbit
			}
		}
	} else if !cfg.InfraOptional {
		if pool != nil {
			pool.Close()
		}
		return nil, errors.New("RABBITMQ_URL is required")
	}

	hasher := infsvc.BcryptHasher{}
	clock := service.SystemClock{}
	taskUC := usecase.TaskUseCase{
		Tasks:     tasks,
		Publisher: publisher,
		Clock:     clock,
	}
	accountUC := usecase.AccountUseCase{
		Accounts: accounts,
		Tokens:   tokens,
		Hasher:   hasher,
		Clock:    clock,
	}
	loginUC := usecase.LoginUseCase{
		Accounts: accounts,
		Tokens:   tokens,
		Hasher:   hasher,
	}
	auth := httpapi.Auth{Accounts: accounts, Tokens: tokens}
	handler := httpapi.Server{
		Handler: httpapi.TaskHandler{UseCase: taskUC, Bundle: bundle},
		Login:   httpapi.LoginHandler{UseCase: loginUC, Bundle: bundle},
		Account: httpapi.AccountHandler{UseCase: accountUC, Bundle: bundle},
		Profile: httpapi.ProfileHandler{UseCase: accountUC, Bundle: bundle},
		Auth:    auth,
		Bundle:  bundle,
		Log:     logApp,
	}.Router()

	ln, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		if pool != nil {
			pool.Close()
		}
		rabbit.Close()
		return nil, err
	}
	srv := &http.Server{Handler: handler, Addr: ln.Addr().String(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		_ = srv.Serve(ln)
	}()
	return &Runtime{HTTP: srv, Pool: pool, Rabbit: rabbit, Shutdown: cfg.ShutdownTimeout}, nil
}

func (rt *Runtime) Addr() string {
	if rt == nil || rt.HTTP == nil {
		return ""
	}
	return rt.HTTP.Addr
}

func (rt *Runtime) Stop(ctx context.Context) error {
	var first error
	if rt.HTTP != nil {
		first = rt.HTTP.Shutdown(ctx)
	}
	rt.Rabbit.Close()
	if rt.Pool != nil {
		rt.Pool.Close()
	}
	return first
}
