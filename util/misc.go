/*
 * Copyright 2022 InfAI (CC SES)
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

package util

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

func PrintInfo(items ...string) {
	hLen := 0
	for i := 0; i < len(items); i++ {
		if l := utf8.RuneCountInString(items[i]); l > hLen {
			hLen = l
		}
	}
	line := strings.Repeat("*", hLen)
	_, _ = fmt.Fprintln(os.Stderr, line)
	for i := 0; i < len(items); i++ {
		if items[i] != "" {
			_, _ = fmt.Fprintln(os.Stderr, items[i])
		}
	}
	_, _ = fmt.Fprintln(os.Stderr, line)
}

func ToJsonStr(v any) (str string) {
	if b, err := json.Marshal(v); err != nil {
		str = err.Error()
	} else {
		str = string(b)
	}
	return
}
