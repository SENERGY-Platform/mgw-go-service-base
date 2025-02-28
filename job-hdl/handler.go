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

package job_hdl

import (
	"context"
	"fmt"
	"github.com/SENERGY-Platform/go-cc-job-handler/ccjh"
	"github.com/SENERGY-Platform/go-service-base/job-hdl/lib"
	"github.com/google/uuid"
	"sort"
	"sync"
	"time"
)

type Handler struct {
	mu        sync.RWMutex
	ctx       context.Context
	ccHandler *ccjh.Handler
	jobs      map[string]*job
}

func New(ctx context.Context, ccHandler *ccjh.Handler) *Handler {
	return &Handler{
		ctx:       ctx,
		ccHandler: ccHandler,
		jobs:      make(map[string]*job),
	}
}

func (h *Handler) Create(_ context.Context, desc string, tFunc TargetFunc) (string, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		if NewInternalErr != nil {
			err = NewInternalErr(err)
		}
		return "", err
	}
	id := uid.String()
	ctx, cf := context.WithCancel(h.ctx)
	j := job{
		tFunc: tFunc,
		ctx:   ctx,
		cFunc: cf,
		Job: lib.Job{
			ID:          id,
			Created:     time.Now().UTC(),
			Description: desc,
		},
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	err = h.ccHandler.Add(&j)
	if err != nil {
		if NewInternalErr != nil {
			err = NewInternalErr(err)
		}
		return "", err
	}
	h.jobs[id] = &j
	return id, nil
}

func (h *Handler) Get(_ context.Context, id string) (lib.Job, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	j, ok := h.jobs[id]
	if !ok {
		err := fmt.Errorf("%s not found", id)
		if NewNotFoundErr != nil {
			err = NewNotFoundErr(err)
		}
		return lib.Job{}, err
	}
	return j.Meta(), nil
}

func (h *Handler) Cancel(_ context.Context, id string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	j, ok := h.jobs[id]
	if !ok {
		err := fmt.Errorf("%s not found", id)
		if NewNotFoundErr != nil {
			err = NewNotFoundErr(err)
		}
		return err
	}
	j.Cancel()
	return nil
}

func (h *Handler) List(_ context.Context, filter lib.JobFilter) ([]lib.Job, error) {
	if filter.Status != "" {
		_, ok := jobStateMap[filter.Status]
		if !ok {
			err := fmt.Errorf("unknown job status '%s'", filter.Status)
			if NewInvalidInputError != nil {
				err = NewInvalidInputError(err)
			}
			return nil, err
		}
	}
	var jobs []lib.Job
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, v := range h.jobs {
		if check(filter, v.Meta()) {
			jobs = append(jobs, v.Meta())
		}
	}
	if filter.SortDesc {
		sort.Slice(jobs, func(i, j int) bool {
			return jobs[i].Created.After(jobs[j].Created)
		})
	} else {
		sort.Slice(jobs, func(i, j int) bool {
			return jobs[i].Created.Before(jobs[j].Created)
		})
	}
	return jobs, nil
}

func (h *Handler) PurgeJobs(_ context.Context, maxAge time.Duration) (int, error) {
	var l []string
	tNow := time.Now().UTC()
	h.mu.RLock()
	for k, v := range h.jobs {
		m := v.Meta()
		if v.IsCanceled() || m.Completed != nil || m.Canceled != nil {
			if tNow.Sub(m.Created) >= maxAge {
				l = append(l, k)
			}
		}
	}
	h.mu.RUnlock()
	h.mu.Lock()
	for _, id := range l {
		delete(h.jobs, id)
	}
	h.mu.Unlock()
	return len(l), nil
}

func check(filter lib.JobFilter, job lib.Job) bool {
	if !filter.Since.IsZero() && !job.Created.After(filter.Since) {
		return false
	}
	if !filter.Until.IsZero() && !job.Created.Before(filter.Until) {
		return false
	}
	switch filter.Status {
	case lib.JobPending:
		if job.Started != nil || job.Canceled != nil || job.Completed != nil {
			return false
		}
	case lib.JobRunning:
		if job.Started == nil || job.Canceled != nil || job.Completed != nil {
			return false
		}
	case lib.JobCanceled:
		if job.Canceled == nil {
			return false
		}
	case lib.JobCompleted:
		if job.Completed == nil {
			return false
		}
	case lib.JobError:
		if job.Completed != nil && job.Error == nil {
			return false
		}
	case lib.JobOK:
		if job.Completed != nil && job.Error != nil {
			return false
		}
	}
	return true
}

var jobStateMap = map[lib.JobStatus]struct{}{
	lib.JobPending:   {},
	lib.JobRunning:   {},
	lib.JobCanceled:  {},
	lib.JobCompleted: {},
	lib.JobError:     {},
	lib.JobOK:        {},
}
