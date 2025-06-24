/*
Copyright 2025 The Ketches Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type Sessioner interface {
	Get(key string) any
	Set(key string, value any)
	SetFromMap(kvs map[string]any)
	Del(key string)
	DelByKeys(keys []string)
}

type sessioner struct {
	Session sessions.Session
}

func NewSessioner(c *gin.Context) Sessioner {
	session := sessions.Default(c)
	session.Options(sessions.Options{
		SameSite: http.SameSiteNoneMode,
		Secure:   false,
		HttpOnly: true,
	})
	return &sessioner{session}
}

func (s *sessioner) Get(key string) any {
	return s.Session.Get(key)
}

func (s *sessioner) Set(key string, value any) {
	s.Session.Set(key, value)
	s.Session.Save()
}

func (s *sessioner) SetFromMap(kvs map[string]any) {
	for key, value := range kvs {
		s.Set(key, value)
	}
	s.Session.Save()
}

func (s *sessioner) Del(key string) {
	s.Session.Delete(key)
	s.Session.Save()
}

func (s *sessioner) DelByKeys(keys []string) {
	for _, key := range keys {
		s.Del(key)
	}
	s.Session.Save()
}

func GetSession(c *gin.Context, key string) any {
	return NewSessioner(c).Get(key)
}

func GetSessionString(c *gin.Context, key string) string {
	session := NewSessioner(c).Get(key)
	if sessionString, ok := session.(string); ok {
		return sessionString
	}
	return ""
}

func SetSession(c *gin.Context, key string, value any) {
	NewSessioner(c).Set(key, value)
}

func SetSessionsFromMap(c *gin.Context, sessions map[string]any) {
	NewSessioner(c).SetFromMap(sessions)
}

func DeleteSession(c *gin.Context, key string) {
	NewSessioner(c).Del(key)
}

func DeleteSessionsByKeys(c *gin.Context, keys []string) {
	NewSessioner(c).DelByKeys(keys)
}
