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
	"fmt"
	"syscall"
	"testing"
	"time"
)

func task(ctx context.Context, num int) chan struct{} {
	done := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println(num, "stopped")
				done <- struct{}{}
				return
			default:
				fmt.Println(num, time.Now().Unix())
				time.Sleep(time.Second)
			}
		}
	}()
	return done
}

func newTask(num int) func() error {
	ctx, cf := context.WithCancel(context.Background())
	ch := task(ctx, num)
	return func() error {
		cf()
		<-ch
		return nil
	}
}

type Task struct {
	dCHan chan struct{}
	sChan chan bool
	num   int
	stop  bool
}

func NewTask(num int) *Task {
	return &Task{
		dCHan: make(chan struct{}, 1),
		sChan: make(chan bool, 1),
		num:   num,
		stop:  true,
	}
}

func (t *Task) Stop() error {
	if !t.stop {
		t.sChan <- true
		<-t.dCHan
	}
	return nil
}

func (t *Task) Start() {
	t.stop = false
	go func() {
		count := 0
		for !t.stop {
			select {
			case t.stop = <-t.sChan:
			default:
				fmt.Println(t.num, time.Now().Unix())
				time.Sleep(time.Second)
				if count >= 5 {
					t.stop = true
				}
				count += 1
			}
		}
		fmt.Println(t.num, "stopped")
		t.dCHan <- struct{}{}
	}()
}

func (t *Task) IsAlive() bool {
	return !t.stop
}

type logger struct {
}

func (l *logger) Warning(arg ...any) {
	fmt.Println(arg...)
}

func (l *logger) Warningf(format string, arg ...any) {
	fmt.Printf(format, arg...)
}

func (l *logger) Error(arg ...any) {
	fmt.Println(arg...)
}

func TestName(t *testing.T) {
	Logger = &logger{}
	wd := New(syscall.SIGINT, syscall.SIGTERM)
	cf1 := newTask(1)
	cf2 := newTask(2)
	t3 := NewTask(3)
	t3.Start()
	t4 := NewTask(4)
	t4.Start()
	wd.RegisterStopFunc(cf1)
	wd.RegisterStopFunc(cf2)
	wd.RegisterStopFunc(t3.Stop)
	wd.RegisterHealthFunc(t3.IsAlive)
	wd.RegisterStopFunc(t4.Stop)
	wd.RegisterHealthFunc(t4.IsAlive)
	//time.Sleep(time.Second * 10)

	//time.Sleep(time.Second * 5)
	wd.Start()
	wd.Trigger()
	fmt.Println(wd.Join())
}
