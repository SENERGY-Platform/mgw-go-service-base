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

package srv_info_hdl

import (
	"github.com/SENERGY-Platform/mgw-go-service-base/srv-info-hdl/lib"
	"runtime"
	"time"
)

type Handler struct {
	name    string
	version string
	started time.Time
}

func New(name, version string) *Handler {
	return &Handler{
		name:    name,
		version: version,
		started: time.Now(),
	}
}

func (h *Handler) GetInfo() lib.SrvInfo {
	var mStats runtime.MemStats
	runtime.ReadMemStats(&mStats)
	return lib.SrvInfo{
		Name:    h.name,
		Version: h.version,
		UpTime:  time.Since(h.started),
		MemStats: lib.MemStats{
			Alloc:      mStats.Alloc,
			AllocTotal: mStats.TotalAlloc,
			SysTotal:   mStats.Sys,
			GCCycles:   mStats.NumGC,
		},
	}
}

func (h *Handler) GetName() string {
	return h.name
}

func (h *Handler) GetVersion() string {
	return h.version
}
