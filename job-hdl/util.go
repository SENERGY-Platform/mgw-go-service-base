/*
 * Copyright 2024 InfAI (CC SES)
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

package job_hdl

import (
	"context"
	"sync"
	"time"
)

type PurgeJobsHandler struct {
	jobHdl   JobHandler
	interval time.Duration
	maxAge   time.Duration
	started  bool
	dChan    chan struct{}
	mu       sync.Mutex
}

func NewPurgeJobsHandler(jobHdl JobHandler, interval, maxAge time.Duration) *PurgeJobsHandler {
	return &PurgeJobsHandler{
		jobHdl:   jobHdl,
		interval: interval,
		maxAge:   maxAge,
		dChan:    make(chan struct{}),
	}
}

func (h *PurgeJobsHandler) Start(ctx context.Context) {
	h.mu.Lock()
	if !h.started {
		h.started = true
		h.mu.Unlock()
		go h.run(ctx)
	} else {
		h.mu.Unlock()
	}
}

func (h *PurgeJobsHandler) Wait() {
	<-h.dChan
}

func (h *PurgeJobsHandler) run(ctx context.Context) {
	timer := time.NewTimer(h.interval)
	loop := true
	for loop {
		select {
		case <-timer.C:
			if Logger != nil {
				Logger.Debugf("purging old jobs ...")
			}
			if n, err := h.jobHdl.PurgeJobs(ctx, h.maxAge); err != nil {
				if Logger != nil {
					Logger.Errorf("purging old jobs failed: %s", err)
				}
			} else {
				if Logger != nil {
					Logger.Debugf("purged '%d' old jobs", n)
				}
			}
			timer.Reset(h.interval)
		case <-ctx.Done():
			loop = false
			break
		}
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	h.dChan <- struct{}{}
	h.mu.Lock()
	h.started = false
	h.mu.Unlock()
}
