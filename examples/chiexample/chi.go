package chiexample

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var l *el_ratio.LeakybuckerLimiter

func main() {

	l = el_ratio.NewLeakyBucketLimiter(1, 5*time.Second)

	r := chi.NewRouter()
	r.Use(Limiter)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	http.ListenAndServe(":3333", r)
}

var channel chan func() = make(chan func(), 1)

func QueueLimiter(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		channel <- func() {
			for len(channel) > 0 && len(channel) == cap(channel) {
				fmt.Print(".")
				time.Sleep(16 * time.Millisecond)
			}

			ctx := r.Context()
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}

	return http.HandlerFunc(fn)
}

func Limiter(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		now := l.Wait()

		fmt.Println(now)
		ctx := r.Context()
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
