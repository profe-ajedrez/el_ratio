package el_ratio

import (
	"sync/atomic"
	"time"
	"unsafe"

	"code.cloudfoundry.org/clock"
)

// state contiene el estado del limiter. El último momento de comprobación
// y por cuanto tiempo esperar para comprobar de nuevo
type state struct {
	lastTick time.Time
	sleepfor time.Duration
}

// LeakybuckerLimiter establece un limitador de procesamiento en base a un número
// de peticiones de proceso por unidad de tiempo definida, usando una versión
// simplificada del algoritmo leaky bucket.
//
// ver: https://www.sciencedirect.com/topics/computer-science/leaky-bucket-algorithm
//
// en esta versión, las peticiones de proceso no son descartadas si el bucket está lleno,
// solo se ponen en espera hasta que haya disponibilidad para procesarlas.
//
// LeakybuckerLimiter esperará a que sea posible ejecutar el siguiente proceso dentro de la unidad de tiempo
// a través de su método Wait()
type LeakybuckerLimiter struct {
	// state guarda una referencia al último momento en que se comprobó si se podía continuar el proceso
	// y cuanto tiempo esperar para continuar
	state unsafe.Pointer

	// timePerActions indica cuantas acciones realizar por la unidad de tiempo, si lo ejemplificamos en el uso de una api REST,
	// refiere a cuantos request procesar por segundo, milisegundo, minuto etc.
	timePerActions time.Duration

	// slack indica cuanto tiempo esperar antes de volver a comprobar si se puede continuar con el proceso.
	slack time.Duration

	// timer es una referencia a algún tipo de reloj. Esta implementación usa el reloj
	// de code.cloudfoundry.org/clock"
	timer clock.Clock
}

// NewLeakybuckerLimiter devuelve un nuevo limitador leaky bucket listo para usar.
// rate indica cuanto procesos ejecutar por unidad de tiempo
// pertime es la unidad de tiempo, generalment time.Second
//
// Ej:
//
//	limiter := NewLeakybuckerLimiter(1, time.Second*2) // limitaremos a 1 proceso por cada 2 segundos
//
//	prev := time.Now()
//
//	for i := 0; i <= 9; i++ {
//		now := l.Now()
//
//		if i > 0 {
//			ellapsed := now.Sub(prev).Round(time.Millisecond * 2)
//			fmt.Println("round: %d  delay: %s ", i, ellapsed) // al ejecutyar debería verse algo así como: "round: 1  delay: 2s"
//		}
//		prev = now
//	}
//
// Ejemplo en un middleware:
//
//	var l = ratio.NewLeakybuckerLimiter(50, 500*time.Millisecond)
//
//	func Limiter(next http.Handler) http.Handler {
//
//		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//			now := l.Now()
//			next.ServeHTTP(w, r)
//		})
//	}
func NewLeakyBucketLimiter(rate int, perTime time.Duration) *LeakybuckerLimiter {
	lim := &LeakybuckerLimiter{
		timePerActions: perTime / time.Duration(rate),
		slack:          10,
		timer:          clock.NewClock(),
	}

	atomic.StorePointer(&lim.state, unsafe.Pointer(&state{
		lastTick: time.Time{},
		sleepfor: 0,
	}))

	return lim
}

// Wait espera a que sea posible ejecutar el siguiente proceso dentro de la unidad de tiempo definida
// parafraseando la definición de leaky bucket, va llenando el tarro con tokenes de ejecución
// a medida que se van usando dentro de la unidad de tioempo
// para que los procesos puedan ir tomando
func (l *LeakybuckerLimiter) Wait() time.Time {
	var (
		newState state
		interval time.Duration
		allowed  bool
	)

	for !allowed {
		now := l.timer.Now()

		prev := atomic.LoadPointer(&l.state)
		old := (*state)(prev)

		newState = state{
			lastTick: now,
			sleepfor: old.sleepfor,
		}

		if old.lastTick.IsZero() {
			allowed = atomic.CompareAndSwapPointer(&l.state, prev, unsafe.Pointer(&newState))
			continue
		}

		newState.sleepfor += l.timePerActions - now.Sub(old.lastTick)

		if newState.sleepfor < l.slack {
			newState.sleepfor = l.slack
		}

		if newState.sleepfor > 0 {
			newState.lastTick = newState.lastTick.Add(newState.sleepfor)
			interval, newState.sleepfor = newState.sleepfor, 0
		}

		allowed = atomic.CompareAndSwapPointer(&l.state, prev, unsafe.Pointer(&newState))
	}

	l.timer.Sleep(interval)

	return newState.lastTick
}
