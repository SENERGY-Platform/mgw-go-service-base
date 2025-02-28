/*
 * Copyright 2023 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package watchdog

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"
)

var Logger interface {
	Warning(arg ...any)
	Warningf(format string, arg ...any)
	Error(arg ...any)
}

type Watchdog struct {
	signals      map[os.Signal]struct{}
	stpFunc      []func() error
	syncChan     chan struct{}
	sigChan      chan os.Signal
	healthChan   chan struct{}
	healthTicker *time.Ticker
	healthCtx    context.Context
	healthCF     context.CancelFunc
	healthWG     sync.WaitGroup
	mu           sync.Mutex
	ec           int
	started      bool
}

func New(signals ...os.Signal) *Watchdog {
	sig := make(map[os.Signal]struct{})
	for _, s := range signals {
		sig[s] = struct{}{}
	}
	ctx, cf := context.WithCancel(context.Background())
	return &Watchdog{
		signals:      sig,
		sigChan:      make(chan os.Signal, 1),
		healthChan:   make(chan struct{}, 1),
		syncChan:     make(chan struct{}),
		healthTicker: time.NewTicker(time.Second),
		healthCtx:    ctx,
		healthCF:     cf,
	}
}

func (w *Watchdog) RegisterHealthFunc(f func() bool) {
	w.healthWG.Add(1)
	w.startHealthcheck(w.healthCtx, f)
}

func (w *Watchdog) RegisterStopFunc(f func() error) {
	w.mu.Lock()
	w.stpFunc = append(w.stpFunc, f)
	w.mu.Unlock()
}

func (w *Watchdog) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started {
		panic("watchdog already started")
	}
	w.started = true
	for sig := range w.signals {
		signal.Notify(w.sigChan, sig)
	}
	go func() {
		defer w.healthTicker.Stop()
		select {
		case sig := <-w.sigChan:
			if Logger != nil {
				Logger.Warningf("caught signal '%s'", sig)
			}
			break
		case <-w.healthChan:
			w.ec = 1
			break
		}
		if Logger != nil {
			Logger.Warning("stopping ...")
		}
		w.healthCF()
		w.healthWG.Wait()
		w.callStopFunc()
	}()
}

func (w *Watchdog) Join() int {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		<-w.syncChan
		return w.ec
	}
	panic("watchdog not started")
}

func (w *Watchdog) Trigger() {
	select {
	case w.healthChan <- struct{}{}:
	default:
	}
}

func (w *Watchdog) startHealthcheck(ctx context.Context, f func() bool) {
	go func() {
		defer w.healthWG.Done()
		for {
			select {
			case <-w.healthTicker.C:
				if !f() {
					select {
					case w.healthChan <- struct{}{}:
					default:
					}
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (w *Watchdog) callStopFunc() {
	w.mu.Lock()
	defer w.mu.Unlock()
	var wg sync.WaitGroup
	wg.Add(len(w.stpFunc))
	for _, f := range w.stpFunc {
		fu := f
		go func() {
			err := fu()
			if err != nil {
				if Logger != nil {
					Logger.Error(err)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	close(w.syncChan)
}
