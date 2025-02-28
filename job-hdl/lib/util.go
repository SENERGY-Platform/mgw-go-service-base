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

package lib

import (
	"context"
	"time"
)

func Await(ctx context.Context, client Api, jID string, delay, httpTimeout time.Duration, logger interface{ Error(arg ...any) }) (Job, error) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	var cfs []context.CancelFunc
	defer func() {
		for _, cf := range cfs {
			cf()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			c, cf := context.WithTimeout(context.Background(), httpTimeout)
			err := client.CancelJob(c, jID)
			if err != nil && logger != nil {
				logger.Error(err)
			}
			cf()
			return Job{}, ctx.Err()
		case <-ticker.C:
			c, cf := context.WithTimeout(context.Background(), httpTimeout)
			cfs = append(cfs, cf)
			j, err := client.GetJob(c, jID)
			if err != nil {
				return Job{}, err
			}
			if j.Completed != nil {
				return j, nil
			}
		}
	}
}
